package handlers

import (
	"net/http"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/service"
)

type ContentBlockHandler struct {
	svc *service.ContentBlockService
}

func NewContentBlockHandler(svc *service.ContentBlockService) *ContentBlockHandler {
	return &ContentBlockHandler{svc: svc}
}

func (h *ContentBlockHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	blocks, err := h.svc.List(r.Context(), userID, r.PathValue("resumeID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, blocks)
}

func (h *ContentBlockHandler) UpdateContent(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var body struct {
		Content string `json:"content"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	err := h.svc.UpdateContent(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("kind"), r.PathValue("id"), body.Content)
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
