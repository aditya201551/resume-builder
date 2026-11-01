package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/jobsearch"
	"resume-builder/backend/internal/service"
)

type JobHandler struct {
	svc       *service.JobService
	discovery *service.JobDiscoveryService
	assistant *service.JobAssistantService
}

func NewJobHandler(svc *service.JobService) *JobHandler { return &JobHandler{svc: svc} }

func (h *JobHandler) WithDiscovery(discovery *service.JobDiscoveryService, assistant *service.JobAssistantService) *JobHandler {
	h.discovery = discovery
	h.assistant = assistant
	return h
}

func (h *JobHandler) Discover(w http.ResponseWriter, r *http.Request) {
	if h.discovery == nil {
		http.Error(w, "job search is unavailable", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var request jobsearch.Request
	if !decodeJobRequest(w, r, &request) {
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	result, err := h.discovery.Discover(r.Context(), userID, request)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, result)
}

func (h *JobHandler) Refine(w http.ResponseWriter, r *http.Request) {
	if h.assistant == nil {
		http.Error(w, "job assistant is unavailable; use the search filters", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var request struct {
		Intent  jobsearch.Intent `json:"intent"`
		Message string           `json:"message"`
		History []jobsearch.Turn `json:"history"`
	}
	if !decodeJobRequest(w, r, &request) {
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	result, err := h.assistant.Refine(r.Context(), userID, request.Intent, request.Message, request.History)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, result)
}

// Search contracts reject unsupported filters rather than silently ignoring
// them. This is also the boundary for model-generated search requests.
func decodeJobRequest(w http.ResponseWriter, r *http.Request, value any) bool {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(value) != nil {
		http.Error(w, "invalid or unsupported search criteria", http.StatusBadRequest)
		return false
	}
	if err := d.Decode(new(any)); !errors.Is(err, io.EOF) {
		http.Error(w, "expected one search request", http.StatusBadRequest)
		return false
	}
	return true
}

var jobIDPattern = regexp.MustCompile(`^job_[0-9a-f]{32}$`)

func (h *JobHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("jobID")
	if !jobIDPattern.MatchString(id) {
		writeError(w, service.ErrNotFound)
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	job, err := h.svc.Get(r.Context(), userID, id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, job)
}
func (h *JobHandler) Save(w http.ResponseWriter, r *http.Request)   { h.save(w, r, true) }
func (h *JobHandler) Unsave(w http.ResponseWriter, r *http.Request) { h.save(w, r, false) }
func (h *JobHandler) save(w http.ResponseWriter, r *http.Request, save bool) {
	id := r.PathValue("jobID")
	if !jobIDPattern.MatchString(id) {
		writeError(w, service.ErrNotFound)
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	var err error
	if save {
		err = h.svc.Save(r.Context(), userID, id)
	} else {
		err = h.svc.Unsave(r.Context(), userID, id)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *JobHandler) Countries(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.svc.Countries())
}
