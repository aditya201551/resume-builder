package handlers

import (
	"net/http"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/repository"
	"resume-builder/backend/internal/service"
)

type ResumeHandler struct {
	svc *service.ResumeService
}

func NewResumeHandler(svc *service.ResumeService) *ResumeHandler {
	return &ResumeHandler{svc: svc}
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

func (h *ResumeHandler) SwitchTemplate(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var body struct {
		TemplateID string `json:"template_id"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	resume, err := h.svc.SwitchTemplate(r.Context(), userID, r.PathValue("resumeID"), body.TemplateID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resume)
}
