package handlers

import (
	"net/http"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/design"
	"resume-builder/backend/internal/service"
)

type ResumeDesignHandler struct {
	svc *service.ResumeDesignService
}

func NewResumeDesignHandler(svc *service.ResumeDesignService) *ResumeDesignHandler {
	return &ResumeDesignHandler{svc: svc}
}

func (h *ResumeDesignHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	d, err := h.svc.Get(r.Context(), userID, r.PathValue("resumeID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *ResumeDesignHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var in design.ResumeDesign
	if !decodeJSON(w, r, &in) {
		return
	}
	d, err := h.svc.Update(r.Context(), userID, r.PathValue("resumeID"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}
