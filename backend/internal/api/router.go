package api

import (
	"net/http"
	"os"
	"path/filepath"

	"resume-builder/backend/internal/api/handlers"
	"resume-builder/backend/internal/api/middleware"
	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/repository"
)

type Handlers struct {
	Auth           *handlers.AuthHandler
	Resume         *handlers.ResumeHandler
	WorkExperience *handlers.EntityHandler[repository.WorkExperienceInput, repository.WorkExperience]
	Education      *handlers.EntityHandler[repository.EducationInput, repository.Education]
	Project        *handlers.EntityHandler[repository.ProjectInput, repository.Project]
	Certification  *handlers.EntityHandler[repository.CertificationInput, repository.Certification]
	Language       *handlers.EntityHandler[repository.LanguageInput, repository.Language]
	MiscEntry      *handlers.EntityHandler[repository.MiscEntryInput, repository.MiscEntry]
	Skill          *handlers.SkillHandler
	CustomSection  *handlers.CustomSectionHandler
	SectionConfig  *handlers.SectionConfigHandler
	Template       *handlers.TemplateHandler
	ResumeDesign   *handlers.ResumeDesignHandler
	DesignSchema   *handlers.DesignSchemaHandler
	Agent          *handlers.AgentHandler
}

func NewRouter(jwtIssuer *auth.JWTIssuer, h Handlers, healthCheck http.HandlerFunc, staticDir string) http.Handler {
	mux := http.NewServeMux()
	protect := middleware.RequireAuth(jwtIssuer)
	handle := func(pattern string, fn http.HandlerFunc) { mux.Handle(pattern, protect(fn)) }

	mux.HandleFunc("GET /api/health", healthCheck)

	mux.HandleFunc("GET /api/auth/{provider}/login", h.Auth.Login)
	mux.HandleFunc("GET /api/auth/{provider}/callback", h.Auth.Callback)
	mux.HandleFunc("POST /api/auth/logout", h.Auth.Logout)
	handle("GET /api/auth/me", h.Auth.Me)

	handle("GET /api/resumes", h.Resume.List)
	handle("POST /api/resumes", h.Resume.Create)
	handle("GET /api/resumes/{resumeID}", h.Resume.Get)
	handle("PATCH /api/resumes/{resumeID}", h.Resume.Update)
	handle("DELETE /api/resumes/{resumeID}", h.Resume.Delete)
	handle("GET /api/resumes/{resumeID}/full", h.Resume.GetFull)
	handle("POST /api/resumes/{resumeID}/duplicate", h.Resume.Duplicate)
	handle("GET /api/resumes/{resumeID}/export/pdf", h.Resume.ExportPDF)
	mux.HandleFunc("GET /api/resumes/{resumeID}/export/data", h.Resume.ExportData)

	registerEntityRoutes(handle, "/api/resumes/{resumeID}/work-experiences", h.WorkExperience)
	registerEntityRoutes(handle, "/api/resumes/{resumeID}/educations", h.Education)
	registerEntityRoutes(handle, "/api/resumes/{resumeID}/projects", h.Project)
	registerEntityRoutes(handle, "/api/resumes/{resumeID}/certifications", h.Certification)
	registerEntityRoutes(handle, "/api/resumes/{resumeID}/languages", h.Language)
	registerEntityRoutes(handle, "/api/resumes/{resumeID}/misc-entries", h.MiscEntry)

	handle("GET /api/resumes/{resumeID}/skill-groups", h.Skill.ListGroups)
	handle("POST /api/resumes/{resumeID}/skill-groups", h.Skill.CreateGroup)
	handle("PATCH /api/resumes/{resumeID}/skill-groups/{groupID}", h.Skill.UpdateGroup)
	handle("DELETE /api/resumes/{resumeID}/skill-groups/{groupID}", h.Skill.DeleteGroup)
	handle("PUT /api/resumes/{resumeID}/skill-groups/reorder", h.Skill.ReorderGroups)
	handle("POST /api/resumes/{resumeID}/skill-groups/{groupID}/items", h.Skill.CreateItem)
	handle("PATCH /api/resumes/{resumeID}/skill-groups/{groupID}/items/{itemID}", h.Skill.UpdateItem)
	handle("DELETE /api/resumes/{resumeID}/skill-groups/{groupID}/items/{itemID}", h.Skill.DeleteItem)
	handle("PUT /api/resumes/{resumeID}/skill-groups/{groupID}/items/reorder", h.Skill.ReorderItems)

	handle("GET /api/resumes/{resumeID}/custom-sections", h.CustomSection.ListSections)
	handle("POST /api/resumes/{resumeID}/custom-sections", h.CustomSection.CreateSection)
	handle("PATCH /api/resumes/{resumeID}/custom-sections/{sectionID}", h.CustomSection.UpdateSection)
	handle("DELETE /api/resumes/{resumeID}/custom-sections/{sectionID}", h.CustomSection.DeleteSection)
	handle("PUT /api/resumes/{resumeID}/custom-sections/reorder", h.CustomSection.ReorderSections)
	handle("POST /api/resumes/{resumeID}/custom-sections/{sectionID}/entries", h.CustomSection.CreateEntry)
	handle("PATCH /api/resumes/{resumeID}/custom-sections/{sectionID}/entries/{entryID}", h.CustomSection.UpdateEntry)
	handle("DELETE /api/resumes/{resumeID}/custom-sections/{sectionID}/entries/{entryID}", h.CustomSection.DeleteEntry)
	handle("PUT /api/resumes/{resumeID}/custom-sections/{sectionID}/entries/reorder", h.CustomSection.ReorderEntries)

	handle("GET /api/resumes/{resumeID}/section-configs", h.SectionConfig.List)
	handle("PATCH /api/resumes/{resumeID}/section-configs/{sectionType}", h.SectionConfig.Update)

	handle("GET /api/templates", h.Template.List)
	handle("GET /api/design-schema", h.DesignSchema.Get)
	handle("GET /api/resumes/{resumeID}/design", h.ResumeDesign.Get)
	handle("PUT /api/resumes/{resumeID}/design", h.ResumeDesign.Update)

	handle("POST /api/resumes/{resumeID}/agent/chat", h.Agent.Chat)

	if staticDir != "" {
		mux.Handle("/", spaFileServer(staticDir))
	}

	return mux
}

func spaFileServer(dir string) http.HandlerFunc {
	fileServer := http.FileServer(http.Dir(dir))
	return func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		fileServer.ServeHTTP(w, r)
	}
}

func registerEntityRoutes[TInput any, TOutput any](handle func(string, http.HandlerFunc), base string, h *handlers.EntityHandler[TInput, TOutput]) {
	handle("GET "+base, h.List)
	handle("POST "+base, h.Create)
	handle("PATCH "+base+"/{id}", h.Update)
	handle("DELETE "+base+"/{id}", h.Delete)
	handle("PUT "+base+"/reorder", h.Reorder)
}
