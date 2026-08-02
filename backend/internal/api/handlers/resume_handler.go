package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/repository"
	"resume-builder/backend/internal/service"
)

type ResumeHandler struct {
	svc             *service.ResumeService
	export          *service.ExportService
	jwtIssuer       *auth.JWTIssuer
	internalBaseURL string
}

func NewResumeHandler(svc *service.ResumeService, export *service.ExportService, jwtIssuer *auth.JWTIssuer, internalBaseURL string) *ResumeHandler {
	return &ResumeHandler{svc: svc, export: export, jwtIssuer: jwtIssuer, internalBaseURL: internalBaseURL}
}

// exportTokenTTL is intentionally short — the token only needs to live long
// enough for the handler to hand it to chromedp and for the headless
// navigation + one data fetch to complete.
const exportTokenTTL = 60 * time.Second

var filenameUnsafeChars = regexp.MustCompile(`[^a-zA-Z0-9-_ ]+`)

func slugifyFilename(label string) string {
	cleaned := strings.TrimSpace(filenameUnsafeChars.ReplaceAllString(label, ""))
	if cleaned == "" {
		return "resume"
	}
	return strings.ReplaceAll(cleaned, " ", "-")
}

// ExportPDF is called by the logged-in user's browser (normal session
// auth). It mints a short-lived, resume-scoped token and drives a headless
// Chrome render of the print-only route, which authenticates with that
// token instead of a session cookie — see ExportData.
func (h *ResumeHandler) ExportPDF(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	resumeID := r.PathValue("resumeID")

	full, err := h.svc.GetFullResume(r.Context(), userID, resumeID)
	if err != nil {
		writeError(w, err)
		return
	}

	token, err := h.jwtIssuer.IssueExportToken(userID, resumeID, exportTokenTTL)
	if err != nil {
		log.Printf("issue export token: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	printURL := fmt.Sprintf("%s/resumes/%s/print?export_token=%s", h.internalBaseURL, resumeID, url.QueryEscape(token))

	pdfBytes, err := h.export.RenderPDF(r.Context(), printURL)
	if err != nil {
		log.Printf("render pdf: %v", err)
		http.Error(w, "failed to generate pdf", http.StatusInternalServerError)
		return
	}

	filename := slugifyFilename(full.Resume.Label) + ".pdf"
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(pdfBytes)
}

// ExportData is fetched by the print-only page itself, loaded inside
// headless Chrome with no session cookie — it authenticates via a one-time
// export_token instead, scoped to exactly the resume that minted it, and
// isn't wrapped by the normal session middleware.
func (h *ResumeHandler) ExportData(w http.ResponseWriter, r *http.Request) {
	resumeID := r.PathValue("resumeID")

	claims, err := h.jwtIssuer.ParseExportToken(r.URL.Query().Get("export_token"))
	if err != nil || claims.ResumeID != resumeID {
		http.Error(w, "invalid or expired export token", http.StatusUnauthorized)
		return
	}

	full, err := h.svc.GetFullResume(r.Context(), claims.UserID, resumeID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, full)
}

func (h *ResumeHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	resumes, err := h.svc.List(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resumes)
}

func (h *ResumeHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var in repository.ResumeMetaInput
	if !decodeJSON(w, r, &in) {
		return
	}
	resume, err := h.svc.Create(r.Context(), userID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resume)
}

func (h *ResumeHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	resume, err := h.svc.Get(r.Context(), userID, r.PathValue("resumeID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resume)
}

func (h *ResumeHandler) GetFull(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	full, err := h.svc.GetFullResume(r.Context(), userID, r.PathValue("resumeID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, full)
}

func (h *ResumeHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var in repository.ResumeMetaInput
	if !decodeJSON(w, r, &in) {
		return
	}
	resume, err := h.svc.UpdateMeta(r.Context(), userID, r.PathValue("resumeID"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resume)
}

func (h *ResumeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	if err := h.svc.Delete(r.Context(), userID, r.PathValue("resumeID")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ResumeHandler) Duplicate(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	resume, err := h.svc.Duplicate(r.Context(), userID, r.PathValue("resumeID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resume)
}
