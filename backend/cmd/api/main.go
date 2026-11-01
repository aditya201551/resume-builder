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
	"resume-builder/backend/internal/jobagent"
	"resume-builder/backend/internal/jobmap"
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

	jobConfig, err := jobmap.LoadConfig()
	if err != nil {
		slog.Error("startup_failed", "operation", "load job search config", "error", err)
		os.Exit(1)
	}
	jobRepository := repository.NewJobRepository(pool)
	jobService := service.NewJobService(jobRepository, jobConfig)
	jobDiscovery := service.NewJobDiscoveryService(jobRepository, jobService)

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

	agentCfg, err := agent.LoadConfig()
	if err != nil {
		slog.Error("startup_failed", "operation", "load agent config", "error", err)
		os.Exit(1)
	}
	aiAgent, err := agent.New(ctx, agentCfg)
	if err != nil {
		slog.Error("startup_failed", "operation", "create agent", "error", err)
		os.Exit(1)
	}
	agentHandler := handlers.NewAgentHandler(aiAgent, resumeService, templateService)
	importAIConfig, err := ai.LoadConfig()
	if err != nil {
		slog.Error("startup_failed", "operation", "load resume import AI config", "error", err)
		os.Exit(1)
	}
	resumeImportService := service.NewResumeImportService(ai.NewPDFClient(importAIConfig), repository.NewResumeImportRepository(pool))
	jobPlanner, err := jobagent.New(ctx, importAIConfig)
	if err != nil {
		slog.Error("startup_failed", "operation", "create job assistant", "error", err)
		os.Exit(1)
	}
	jobAssistant := service.NewJobAssistantService(jobPlanner, jobRepository)

	h := api.Handlers{
		Auth:           handlers.NewAuthHandler(providers, stateSigner, jwtIssuer, authService, users, cfg.FrontendURL, cfg.CookieSecure(), cfg.JWTTTL),
		Resume:         handlers.NewResumeHandler(resumeService, exportService, jwtIssuer, cfg.InternalBaseURL),
		ResumeImport:   handlers.NewResumeImportHandler(resumeImportService),
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
		Jobs:           handlers.NewJobHandler(jobService).WithDiscovery(jobDiscovery, jobAssistant),
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
