package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"resume-builder/backend/internal/agent"
	"resume-builder/backend/internal/ai"
	"resume-builder/backend/internal/api"
	"resume-builder/backend/internal/api/handlers"
	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/config"
	"resume-builder/backend/internal/db"
	"resume-builder/backend/internal/features"
	flagpostgres "resume-builder/backend/internal/features/postgres"
	"resume-builder/backend/internal/logging"
	"resume-builder/backend/internal/repository"
	"resume-builder/backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err := logging.Configure(); err != nil {
		slog.Error("logging_configuration_failed", "error", err)
		os.Exit(1)
	}
	if err != nil {
		slog.Error("startup_failed", "operation", "load config", "error", err)
		os.Exit(1)
	}

	slog.Info("database_migrations_started")
	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		slog.Error("startup_failed", "operation", "run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database_migrations_completed")

	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("startup_failed", "operation", "connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("database_pool_initialized")

	providers := auth.NewProviders(cfg.OAuthRedirectBaseURL, cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GithubClientID, cfg.GithubClientSecret)
	if len(providers) == 0 {
		slog.Warn("oauth_providers_not_configured")
	}

	jwtIssuer := auth.NewJWTIssuer(cfg.JWTSecret, cfg.JWTTTL)
	stateSigner := auth.NewStateSigner(cfg.JWTSecret, 10*time.Minute)

	users := repository.NewUserRepository(pool)
	resumes := repository.NewResumeRepository(pool)
	workExperiences := repository.NewWorkExperienceRepository(pool)
	educations := repository.NewEducationRepository(pool)
	skills := repository.NewSkillRepository(pool)
	projects := repository.NewProjectRepository(pool)
	certifications := repository.NewCertificationRepository(pool)
	languages := repository.NewLanguageRepository(pool)
	miscEntries := repository.NewMiscEntryRepository(pool)
	customSections := repository.NewCustomSectionRepository(pool)
	sectionConfigs := repository.NewSectionConfigRepository(pool)
	resumeDesigns := repository.NewResumeDesignRepository(pool)

	authService := service.NewAuthService(users)
	resumeService := service.NewResumeService(resumes, resumeDesigns, workExperiences, educations, skills, projects, certifications, languages, miscEntries, customSections, sectionConfigs)
	workExperienceService := service.NewWorkExperienceService(resumes, workExperiences)
	educationService := service.NewEducationService(resumes, educations)
	projectService := service.NewProjectService(resumes, projects)
	certificationService := service.NewCertificationService(resumes, certifications)
	languageService := service.NewLanguageService(resumes, languages)
	miscEntryService := service.NewMiscEntryService(resumes, miscEntries)
	skillService := service.NewSkillService(resumes, skills)
	customSectionService := service.NewCustomSectionService(resumes, sectionConfigs, customSections)
	sectionConfigService := service.NewSectionConfigService(resumes, sectionConfigs)
	templateService := service.NewTemplateService()
	resumeDesignService := service.NewResumeDesignService(resumes, resumeDesigns)
	exportService := service.NewExportService(cfg.ChromeExecPath)

	featureService := features.NewService(flagpostgres.NewStore(pool))
	platformConfig, err := ai.LoadConfig()
	if err != nil {
		platformConfig = nil
		slog.Warn("platform_ai_not_configured")
	}
	if platformConfig != nil && platformConfig.Provider == ai.ProviderOpenAI && os.Getenv("AGENT_MODEL") != "" && !ai.CompatibleOpenAI(os.Getenv("AGENT_MODEL")) {
		platformConfig = nil
		slog.Warn("platform_agent_model_unsupported")
	}
	vault, err := ai.LoadVault()
	if err != nil {
		vault = nil
		slog.Error("byok_encryption_not_configured")
	}
	maxTokens, err := agent.LoadMaxTokens()
	if err != nil {
		slog.Error("startup_failed", "operation", "load agent token limit", "error", err)
		os.Exit(1)
	}
	aiSettingsService := service.NewAISettingsService(repository.NewAISettingsRepository(pool), vault, platformConfig, os.Getenv("AGENT_MODEL"))
	aiHandler := handlers.NewAIHandler(aiSettingsService, featureService, cfg.FrontendURL)
	agentHandler := handlers.NewAgentHandler(aiHandler, maxTokens, resumeService, templateService)
	resumeImportService := service.NewResumeImportService(repository.NewResumeImportRepository(pool))

	h := api.Handlers{
		AI:             aiHandler,
		Auth:           handlers.NewAuthHandler(providers, stateSigner, jwtIssuer, authService, users, cfg.FrontendURL, cfg.CookieSecure(), cfg.JWTTTL),
		Resume:         handlers.NewResumeHandler(resumeService, exportService, jwtIssuer, cfg.InternalBaseURL),
		ResumeImport:   handlers.NewResumeImportHandler(resumeImportService, aiHandler),
		WorkExperience: handlers.NewEntityHandler(workExperienceService),
		Education:      handlers.NewEntityHandler(educationService),
		Project:        handlers.NewEntityHandler(projectService),
		Certification:  handlers.NewEntityHandler(certificationService),
		Language:       handlers.NewEntityHandler(languageService),
		MiscEntry:      handlers.NewEntityHandler(miscEntryService),
		Skill:          handlers.NewSkillHandler(skillService),
		CustomSection:  handlers.NewCustomSectionHandler(customSectionService),
		SectionConfig:  handlers.NewSectionConfigHandler(sectionConfigService),
		Template:       handlers.NewTemplateHandler(templateService),
		ResumeDesign:   handlers.NewResumeDesignHandler(resumeDesignService),
		DesignSchema:   handlers.NewDesignSchemaHandler(),
		Agent:          agentHandler,
		Features:       handlers.NewFeatureHandler(featureService),
	}

	healthCheck := func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			if r.Context().Err() != nil {
				slog.Info("health_request_canceled")
				return
			}
			slog.ErrorContext(r.Context(), "health_database_failed", "error", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"db_unreachable"}`))
			return
		}
		w.Write([]byte(`{"status":"ok"}`))
	}

	router := api.NewRouter(jwtIssuer, h, healthCheck, cfg.StaticDir)

	slog.Info("api_listening", "port", cfg.Port)
	server := &http.Server{Addr: ":" + cfg.Port, Handler: router, ErrorLog: slog.NewLogLogger(slog.Default().Handler(), slog.LevelError)}
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.ListenAndServe() }()
	failed := false
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("api_server_failed", "error", err)
			failed = true
		}
	case <-ctx.Done():
		slog.Info("api_shutdown_started")
	}
	stopSignals()
	shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		slog.Warn("api_shutdown_timeout", "error", err)
		if closeErr := server.Close(); closeErr != nil {
			slog.Error("api_close_failed", "error", closeErr)
		}
	}
	slog.Info("api_shutdown_completed")
	if failed {
		os.Exit(1)
	}
}
