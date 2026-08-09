package handlers

import (
	"net/http"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/service"
)

type EntityHandler[TInput any, TOutput any] struct {
	svc service.EntityService[TInput, TOutput]
}

func NewEntityHandler[TInput any, TOutput any](svc service.EntityService[TInput, TOutput]) *EntityHandler[TInput, TOutput] {
	return &EntityHandler[TInput, TOutput]{svc: svc}
}

func (h *EntityHandler[TInput, TOutput]) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	items, err := h.svc.List(r.Context(), userID, r.PathValue("resumeID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *EntityHandler[TInput, TOutput]) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var in TInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.svc.Create(r.Context(), userID, r.PathValue("resumeID"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *EntityHandler[TInput, TOutput]) Update(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var in TInput
	if !decodeJSON(w, r, &in) {
		return
	}
	out, err := h.svc.Update(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("id"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *EntityHandler[TInput, TOutput]) Delete(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	if err := h.svc.Delete(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *EntityHandler[TInput, TOutput]) Reorder(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var body struct {
		OrderedIDs []string `json:"ordered_ids"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := h.svc.Reorder(r.Context(), userID, r.PathValue("resumeID"), body.OrderedIDs); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
