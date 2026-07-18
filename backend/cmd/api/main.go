package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"resume-builder/backend/internal/api"
	"resume-builder/backend/internal/api/handlers"
	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/config"
	"resume-builder/backend/internal/db"
	"resume-builder/backend/internal/repository"
	"resume-builder/backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	providers := auth.NewProviders(cfg.OAuthRedirectBaseURL, cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GithubClientID, cfg.GithubClientSecret)
	if len(providers) == 0 {
		log.Println("warning: no SSO providers configured (set GOOGLE_CLIENT_ID/SECRET or GITHUB_CLIENT_ID/SECRET in .env)")
	}

	jwtIssuer := auth.NewJWTIssuer(cfg.JWTSecret, cfg.JWTTTL)
	stateSigner := auth.NewStateSigner(cfg.JWTSecret, 10*time.Minute)

	// Repositories
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
	templates := repository.NewTemplateRepository(pool)
	contentBlocks := repository.NewContentBlockRepository(pool, workExperiences, projects)

	// Services
	authService := service.NewAuthService(users)
	resumeService := service.NewResumeService(resumes, workExperiences, educations, skills, projects, certifications, languages, miscEntries, customSections, sectionConfigs)
	workExperienceService := service.NewWorkExperienceService(resumes, workExperiences)
	educationService := service.NewEducationService(resumes, educations)
	projectService := service.NewProjectService(resumes, projects)
	certificationService := service.NewCertificationService(resumes, certifications)
	languageService := service.NewLanguageService(resumes, languages)
	miscEntryService := service.NewMiscEntryService(resumes, miscEntries)
	skillService := service.NewSkillService(resumes, skills)
	customSectionService := service.NewCustomSectionService(resumes, sectionConfigs, customSections)
	sectionConfigService := service.NewSectionConfigService(resumes, sectionConfigs)
	templateService := service.NewTemplateService(templates)
	contentBlockService := service.NewContentBlockService(resumes, contentBlocks)
	exportService := service.NewExportService(cfg.ChromeExecPath)

	// Handlers
	h := api.Handlers{
		Auth:           handlers.NewAuthHandler(providers, stateSigner, jwtIssuer, authService, users, cfg.FrontendURL, cfg.CookieSecure(), cfg.JWTTTL),
		Resume:         handlers.NewResumeHandler(resumeService, exportService, jwtIssuer, cfg.FrontendURL),
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
		ContentBlock:   handlers.NewContentBlockHandler(contentBlockService),
	}

	healthCheck := func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"db_unreachable"}`))
			return
		}
		w.Write([]byte(`{"status":"ok"}`))
	}

	router := api.NewRouter(jwtIssuer, h, healthCheck)

	log.Printf("api listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatal(err)
	}
}
