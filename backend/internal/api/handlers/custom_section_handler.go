package handlers

import (
	"net/http"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/repository"
	"resume-builder/backend/internal/service"
)

type CustomSectionHandler struct {
	svc *service.CustomSectionService
}

func NewCustomSectionHandler(svc *service.CustomSectionService) *CustomSectionHandler {
	return &CustomSectionHandler{svc: svc}
}

func (h *CustomSectionHandler) ListSections(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	sections, err := h.svc.ListSections(r.Context(), userID, r.PathValue("resumeID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sections)
}

func (h *CustomSectionHandler) CreateSection(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var in repository.CustomSectionInput
	if !decodeJSON(w, r, &in) {
		return
	}
	s, err := h.svc.CreateSection(r.Context(), userID, r.PathValue("resumeID"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

func (h *CustomSectionHandler) UpdateSection(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var in repository.CustomSectionInput
	if !decodeJSON(w, r, &in) {
		return
	}
	s, err := h.svc.UpdateSection(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("sectionID"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *CustomSectionHandler) DeleteSection(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	if err := h.svc.DeleteSection(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("sectionID")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CustomSectionHandler) ReorderSections(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var body struct {
		OrderedIDs []string `json:"ordered_ids"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := h.svc.ReorderSections(r.Context(), userID, r.PathValue("resumeID"), body.OrderedIDs); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CustomSectionHandler) CreateEntry(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var in repository.CustomSectionEntryInput
	if !decodeJSON(w, r, &in) {
		return
	}
	e, err := h.svc.CreateEntry(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("sectionID"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *CustomSectionHandler) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var in repository.CustomSectionEntryInput
	if !decodeJSON(w, r, &in) {
		return
	}
	e, err := h.svc.UpdateEntry(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("sectionID"), r.PathValue("entryID"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (h *CustomSectionHandler) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	if err := h.svc.DeleteEntry(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("sectionID"), r.PathValue("entryID")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CustomSectionHandler) ReorderEntries(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var body struct {
		OrderedIDs []string `json:"ordered_ids"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := h.svc.ReorderEntries(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("sectionID"), body.OrderedIDs); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
