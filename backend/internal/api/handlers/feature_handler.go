package handlers

import (
	"net/http"

	"resume-builder/backend/internal/features"
	"resume-builder/backend/internal/platform/httpresponse"
)

type FeatureHandler struct{ service *features.Service }

func NewFeatureHandler(service *features.Service) *FeatureHandler {
	return &FeatureHandler{service: service}
}

func (h *FeatureHandler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	httpresponse.JSON(w, http.StatusOK, h.service.Snapshot(r.Context()))
}
