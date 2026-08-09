package handlers

import (
	"net/http"

	"resume-builder/backend/internal/design"
)

type DesignSchemaHandler struct{}

func NewDesignSchemaHandler() *DesignSchemaHandler {
	return &DesignSchemaHandler{}
}

func (h *DesignSchemaHandler) Get(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, design.Schema())
}
