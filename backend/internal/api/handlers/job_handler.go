package handlers

import (
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/jobmap"
	"resume-builder/backend/internal/service"
)

type JobHandler struct{ svc *service.JobService }

func NewJobHandler(svc *service.JobService) *JobHandler { return &JobHandler{svc: svc} }

func (h *JobHandler) Search(w http.ResponseWriter, r *http.Request) {
	s, err := ParseJobSearch(r)
	if err != nil {
		writeError(w, fmt.Errorf("%s: %w", err, service.ErrValidation))
		return
	}
	userID, _ := auth.UserIDFromContext(r.Context())
	result, err := h.svc.Search(r.Context(), userID, s)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, result)
}

func ParseJobSearch(r *http.Request) (jobmap.Search, error) {
	v := r.URL.Query()
	s := jobmap.Search{Query: strings.TrimSpace(v.Get("q")), Workplace: v.Get("workplace"), EmploymentType: v.Get("employment"), Seniority: v.Get("seniority"), SourceKind: v.Get("source"), Country: v.Get("country"), RemoteCountry: v.Get("remoteCountry"), IncludeUnknown: v.Get("includeUnknown") == "true", SavedOnly: v.Get("saved") == "true", Page: 1, PageSize: 50}
	if len([]rune(s.Query)) > 160 {
		return s, fmt.Errorf("search text is too long")
	}
	for value, allowed := range map[string][]string{
		"workplace":  {"", "all", "onsite", "hybrid", "remote", "unknown"},
		"employment": {"", "full_time", "part_time", "contract", "internship", "unknown"},
		"seniority":  {"", "junior", "mid", "senior", "lead", "unknown"},
		"source":     {"", "company", "aggregator", "career_page"},
	} {
		found := false
		for _, a := range allowed {
			if v.Get(value) == a {
				found = true
				break
			}
		}
		if !found {
			return s, fmt.Errorf("invalid %s filter", value)
		}
	}
	for name, target := range map[string]*string{"country": &s.Country, "remoteCountry": &s.RemoteCountry} {
		if *target != "" {
			country, ok := jobmap.FindCountry(*target)
			if !ok {
				return s, fmt.Errorf("invalid %s", name)
			}
			*target = country.Code
		}
	}
	for name, target := range map[string]*int{"page": &s.Page, "pageSize": &s.PageSize, "days": &s.PostedDays} {
		if raw := v.Get(name); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 || n > 100000 {
				return s, fmt.Errorf("invalid %s", name)
			}
			*target = n
		}
	}
	if s.Page < 1 || s.PageSize < 1 || s.PageSize > 100 || s.PostedDays > 365 {
		return s, fmt.Errorf("page, page size or posting age is out of range")
	}
	if raw := v.Get("bounds"); raw != "" {
		parts := strings.Split(raw, ",")
		if len(parts) != 4 {
			return s, fmt.Errorf("bounds needs west,south,east,north")
		}
		b := [4]float64{}
		for i, p := range parts {
			n, err := strconv.ParseFloat(p, 64)
			if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
				return s, fmt.Errorf("invalid map bounds")
			}
			b[i] = n
		}
		if b[0] < -180 || b[0] > 180 || b[2] < -180 || b[2] > 180 || b[1] < -90 || b[3] > 90 || b[1] >= b[3] {
			return s, fmt.Errorf("map bounds are out of range")
		}
		s.Bounds = &b
	}
	return s, nil
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
func (h *JobHandler) Places(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) > 160 {
		writeError(w, fmt.Errorf("location text is too long: %w", service.ErrValidation))
		return
	}
	items, err := h.svc.Places(r.Context(), q)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
func (h *JobHandler) Nearest(w http.ResponseWriter, r *http.Request) {
	lat, e1 := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, e2 := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if e1 != nil || e2 != nil || math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		writeError(w, fmt.Errorf("invalid coordinates: %w", service.ErrValidation))
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, h.svc.Nearest(lat, lng))
}
