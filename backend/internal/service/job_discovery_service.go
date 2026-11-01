package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"resume-builder/backend/internal/jobmap"
	"resume-builder/backend/internal/jobsearch"
)

type discoveryStore interface {
	DiscoveryPage(context.Context, string, string) (jobsearch.Page, bool, error)
	StoreDiscoveryPage(context.Context, string, jobsearch.Page) error
	SavedDiscoveryPage(context.Context, string, int) ([]jobmap.Job, bool, error)
	ClaimExternalSearch(context.Context, string, string, time.Duration, time.Duration) (bool, error)
	CompleteExternalSearch(context.Context, string, string) error
	ReleaseExternalSearch(context.Context, string, string) error
	FailExternalSearch(context.Context, string, string, time.Time) error
	ReserveRequest(context.Context, string, string, int) (bool, error)
}

// JobDiscoveryService bounds work to one page per requested provider/branch.
// The old global job index is deliberately not used for discovery.
type JobDiscoveryService struct {
	repo discoveryStore
	jobs *JobService
	gate chan struct{}
}

func NewJobDiscoveryService(repo discoveryStore, jobs *JobService) *JobDiscoveryService {
	return &JobDiscoveryService{repo: repo, jobs: jobs, gate: make(chan struct{}, 4)}
}

type discoveryCursor struct {
	Key       string             `json:"key"`
	Positions []providerPosition `json:"positions"`
}
type providerPosition struct {
	Provider string `json:"provider"`
	Branch   int    `json:"branch"`
	Page     int    `json:"page"`
	Cursor   string `json:"cursor"`
	Done     bool   `json:"done"`
}

func initialPositions(s jobsearch.Intent) []providerPosition {
	states := []providerPosition{}
	for i, b := range s.Branches {
		if b.Workplace == "remote" || b.Workplace == "all" && b.Country == "" {
			states = append(states, providerPosition{Provider: "himalayas", Branch: i, Page: 1})
		}
		if b.Workplace != "remote" {
			states = append(states, providerPosition{Provider: "jsearch", Branch: i, Page: 1})
		}
	}
	return states
}

func (s *JobDiscoveryService) Discover(ctx context.Context, userID string, r jobsearch.Request) (jobsearch.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	result := jobsearch.Result{Jobs: []jobmap.Job{}, Sources: []jobsearch.Source{}}
	if userID == "" {
		return result, ErrNotFound
	}
	if err := r.Intent.Validate(); err != nil {
		return result, fmt.Errorf("%s: %w", err, ErrValidation)
	}
	if r.SavedOnly {
		return s.saved(ctx, userID, r)
	}
	state := discoveryCursor{Key: r.Intent.Key(), Positions: initialPositions(r.Intent)}
	if r.NextToken != "" {
		data, err := base64.RawURLEncoding.DecodeString(r.NextToken)
		if err != nil || len(data) > 24000 {
			return result, fmt.Errorf("invalid search continuation: %w", ErrValidation)
		}
		var next discoveryCursor
		if json.Unmarshal(data, &next) != nil || next.Key != state.Key || len(next.Positions) != len(state.Positions) {
			return result, fmt.Errorf("search changed; start from the first page: %w", ErrValidation)
		}
		for i, p := range next.Positions {
			expected := state.Positions[i]
			if p.Provider != expected.Provider || p.Branch != expected.Branch || p.Page < 1 || p.Page > 51 || len(p.Cursor) > 2048 {
				return result, fmt.Errorf("invalid provider continuation: %w", ErrValidation)
			}
		}
		state = next
	}
	seen := map[string]bool{}
	for i := range state.Positions {
		p := &state.Positions[i]
		if p.Done {
			continue
		}
		status := jobsearch.Source{ID: fmt.Sprintf("%s:%d", p.Provider, p.Branch), Name: map[string]string{"himalayas": "Himalayas", "jsearch": "JSearch"}[p.Provider]}
		if p.Page > 50 {
			p.Done = true
			status.Status = "limit"
			status.Message = "Refine this search to explore further results."
			result.Sources = append(result.Sources, status)
			continue
		}
		b := r.Intent.Branches[p.Branch]
		if p.Provider == "jsearch" && (s.jobs.cfg.JSearchKey == "" || b.Country == "") {
			p.Done = true
			status.Status = "needs_location"
			status.Message = "Choose an office country for broad job search. Remote discovery remains worldwide."
			if s.jobs.cfg.JSearchKey == "" {
				status.Status = "needs_key"
				status.Message = "Broad search is not connected. Remote search and saved jobs remain available."
			}
			result.Sources = append(result.Sources, status)
			continue
		}
		// Local eligibility filters do not cause another paid upstream request.
		// Share identical provider queries across users and location alternatives.
		queryParts, _ := json.Marshal(struct {
			Provider, Query, Workplace, Country, City, Cursor string
			Page                                              int
		}{p.Provider, strings.ToLower(r.Intent.Query), b.Workplace, b.Country, strings.ToLower(b.City), p.Cursor, p.Page})
		key := string(queryParts)
		// Cache keys are hashes; provider cursors never appear in logs or SQL text.
		key = jobmap.StableID("discovery-v1", key)
		page, found, err := s.repo.DiscoveryPage(ctx, userID, key)
		if err != nil {
			return result, err
		}
		if !found {
			page, status.Status, status.Message, err = s.fetchPage(ctx, key, r.Intent, *p)
			if err != nil {
				if ctx.Err() != nil {
					return result, ctx.Err()
				}
				logJobError(ctx, "job_discovery_failed", err, "provider", p.Provider)
				status.Status = "error"
				status.Message = "This source is temporarily unavailable. Retry this search; other results remain available."
			}
			if status.Status != "ready" {
				result.Retry = true
				result.Sources = append(result.Sources, status)
				continue
			}
			// Join the current user's bookmarks, even for newly populated shared pages.
			page, _, err = s.repo.DiscoveryPage(ctx, userID, key)
			if err != nil {
				return result, err
			}
		}
		status.Status = "ready"
		status.FetchedAt = &page.FetchedAt
		result.Sources = append(result.Sources, status)
		for _, j := range page.Jobs {
			if !r.Intent.Matches(j, time.Now().UTC(), false) {
				continue
			}
			identity := jobmap.ApplicationKey(j.ApplyURL)
			if identity == "" {
				identity = j.ID
			}
			if seen[identity] {
				continue
			}
			seen[identity] = true
			j = discoverySummary(j)
			result.Jobs = append(result.Jobs, j)
		}
		p.Cursor = page.Cursor
		p.Page++
		p.Done = !page.HasMore
	}
	for _, p := range state.Positions {
		if !p.Done {
			data, _ := json.Marshal(state)
			result.NextToken = base64.RawURLEncoding.EncodeToString(data)
			break
		}
	}
	return result, nil
}

func (s *JobDiscoveryService) fetchPage(ctx context.Context, key string, intent jobsearch.Intent, p providerPosition) (jobsearch.Page, string, string, error) {
	page := jobsearch.Page{Jobs: []jobmap.Job{}}
	select {
	case s.gate <- struct{}{}:
		defer func() { <-s.gate }()
	default:
		return page, "busy", "Search is busy. Retry in a moment.", nil
	}
	token := rand.Text()
	// The page table owns freshness. A completion marker must not prevent an
	// evicted page from being fetched again.
	claimed, err := s.repo.ClaimExternalSearch(ctx, "discovery:"+key, token, time.Minute, 0)
	if err != nil {
		return page, "", "", err
	}
	if !claimed {
		return page, "busy", "This search is being prepared or is waiting to retry.", nil
	}
	defer func() {
		cleanup, cancel := jobCleanupContext(ctx)
		defer cancel()
		_ = s.repo.ReleaseExternalSearch(cleanup, "discovery:"+key, token)
	}()
	work, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	b := intent.Branches[p.Branch]
	var jobs []jobmap.Job
	if p.Provider == "himalayas" {
		if err = s.jobs.reserveHimalayasRequest(work); err == nil {
			jobs, page.HasMore, err = s.jobs.client.SearchHimalayasPage(work, intent.Query, b.Country, p.Page)
		}
		if err != nil {
			s.jobs.himalayasBackoff(ctx, err)
		}
	} else {
		var allowed bool
		allowed, err = s.repo.ReserveRequest(work, "jsearch", time.Now().UTC().Format("2006-01"), s.jobs.cfg.JSearchLimit)
		if err == nil && !allowed {
			return page, "quota_exhausted", "Broad search has reached its configured monthly budget. Cached pages and saved jobs remain available.", nil
		}
		query := intent.Query
		if query == "" {
			query = "technology jobs"
		}
		if b.Workplace != "all" && b.Workplace != "unknown" {
			query += " " + b.Workplace
		}
		if b.City != "" {
			query += " in " + b.City
		}
		if c, ok := jobmap.FindCountry(b.Country); ok {
			query += " in " + c.Name
		}
		if err == nil {
			jobs, page.Cursor, err = s.jobs.client.FetchJSearchPage(work, query, b.Country, p.Cursor)
			page.HasMore = page.Cursor != ""
		}
	}
	if err != nil {
		cleanup, cancel := jobCleanupContext(ctx)
		defer cancel()
		_ = s.repo.FailExternalSearch(cleanup, "discovery:"+key, token, time.Now().Add(time.Minute))
		return page, "", "", err
	}
	if len(jobs) > 100 || len(page.Cursor) > 2048 {
		return page, "", "", fmt.Errorf("provider page exceeds safe bounds")
	}
	page.FetchedAt = time.Now().UTC()
	for _, j := range jobs {
		if prepared, ok := jobmap.Prepare(j, page.FetchedAt); ok {
			page.Jobs = append(page.Jobs, prepared)
		}
	}
	if err = s.repo.StoreDiscoveryPage(work, key, page); err != nil {
		return page, "", "", err
	}
	if err = s.repo.CompleteExternalSearch(work, "discovery:"+key, token); err != nil {
		return page, "", "", err
	}
	return page, "ready", "", nil
}

func (s *JobDiscoveryService) saved(ctx context.Context, userID string, r jobsearch.Request) (jobsearch.Result, error) {
	result := jobsearch.Result{Jobs: []jobmap.Job{}, Sources: []jobsearch.Source{}}
	offset := 0
	if r.NextToken != "" {
		parts := strings.Split(r.NextToken, ":")
		if len(parts) != 3 || parts[0] != "saved" || parts[1] != r.Intent.Key() {
			return result, fmt.Errorf("invalid saved continuation: %w", ErrValidation)
		}
		var err error
		offset, err = strconv.Atoi(parts[2])
		if err != nil || offset < 0 || offset > 100000 {
			return result, ErrValidation
		}
	}
	jobs, more, err := s.repo.SavedDiscoveryPage(ctx, userID, offset)
	if err != nil {
		return result, err
	}
	for _, j := range jobs {
		if r.Intent.Matches(j, time.Now().UTC(), true) {
			result.Jobs = append(result.Jobs, discoverySummary(j))
		}
	}
	if more {
		result.NextToken = "saved:" + r.Intent.Key() + ":" + strconv.Itoa(offset+50)
	}
	return result, nil
}

func discoverySummary(j jobmap.Job) jobmap.Job {
	runes := []rune(j.Description)
	if len(runes) > 320 {
		j.Description = string(runes[:320]) + "…"
	}
	return j
}
