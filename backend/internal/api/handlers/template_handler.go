package handlers

import (
	"net/http"

	"resume-builder/backend/internal/service"
)

type TemplateHandler struct {
	svc *service.TemplateService
}

func NewTemplateHandler(svc *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{svc: svc}
}

func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	templates, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, templates)
}
