package handlers

import (
	"net/http"

	"resume-builder/backend/internal/design"
)

// DesignSchemaHandler serves the static, global field schema every
// template's config UI is rendered from — see design.Schema()'s comment.
// No service/repo behind it: the schema is code, not data.
type DesignSchemaHandler struct{}

func NewDesignSchemaHandler() *DesignSchemaHandler {
	return &DesignSchemaHandler{}
}

func (h *DesignSchemaHandler) Get(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, design.Schema())
}
