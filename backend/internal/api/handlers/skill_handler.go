package handlers

import (
	"net/http"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/repository"
	"resume-builder/backend/internal/service"
)

type SkillHandler struct {
	svc *service.SkillService
}

func NewSkillHandler(svc *service.SkillService) *SkillHandler {
	return &SkillHandler{svc: svc}
}

func (h *SkillHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	groups, err := h.svc.ListGroups(r.Context(), userID, r.PathValue("resumeID"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (h *SkillHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var in repository.SkillGroupInput
	if !decodeJSON(w, r, &in) {
		return
	}
	g, err := h.svc.CreateGroup(r.Context(), userID, r.PathValue("resumeID"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (h *SkillHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var in repository.SkillGroupInput
	if !decodeJSON(w, r, &in) {
		return
	}
	g, err := h.svc.UpdateGroup(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("groupID"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (h *SkillHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	if err := h.svc.DeleteGroup(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("groupID")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SkillHandler) ReorderGroups(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var body struct {
		OrderedIDs []string `json:"ordered_ids"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := h.svc.ReorderGroups(r.Context(), userID, r.PathValue("resumeID"), body.OrderedIDs); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SkillHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var in repository.SkillItemInput
	if !decodeJSON(w, r, &in) {
		return
	}
	it, err := h.svc.CreateItem(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("groupID"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, it)
}

func (h *SkillHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var in repository.SkillItemInput
	if !decodeJSON(w, r, &in) {
		return
	}
	it, err := h.svc.UpdateItem(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("groupID"), r.PathValue("itemID"), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, it)
}

func (h *SkillHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	if err := h.svc.DeleteItem(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("groupID"), r.PathValue("itemID")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SkillHandler) ReorderItems(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.UserIDFromContext(r.Context())
	var body struct {
		OrderedIDs []string `json:"ordered_ids"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := h.svc.ReorderItems(r.Context(), userID, r.PathValue("resumeID"), r.PathValue("groupID"), body.OrderedIDs); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
