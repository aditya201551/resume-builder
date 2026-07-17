package handlers

import (
	"net/http"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/repository"
	"resume-builder/backend/internal/service"
)

type SectionConfigHandler struct {
	svc *service.SectionConfigService
}

func NewSectionConfigHandler(svc *service.SectionConfigService) *SectionConfigHandler {
	return &SectionConfigHandler{svc: svc}
}

func (h *SectionConfigHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	configs, err := h.svc.List(r.Context(), userID, r.PathValue("resumeID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, configs)
}

// Update is keyed by (section_type, custom_section_id) via the path segment
// plus an optional query param, since that pair — not a synthetic row id —
// is the config's real identity (see the schema's UNIQUE constraint).
func (h *SectionConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var in repository.SectionConfigInput
	if !decodeJSON(w, r, &in) {
		return
	}

	var customSectionID *string
	if v := r.URL.Query().Get("custom_section_id"); v != "" {
		customSectionID = &v
	}

	c, err := h.svc.Update(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("sectionType"), customSectionID, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}
