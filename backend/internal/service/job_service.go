package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"resume-builder/backend/internal/jobmap"
	"resume-builder/backend/internal/repository"
)

type jobStore interface {
	StoreSource(context.Context, string, []jobmap.Job, bool) error
	List(context.Context, string, bool) ([]jobmap.Job, error)
	Get(context.Context, string, string) (jobmap.Job, error)
	Save(context.Context, string, string) error
	Unsave(context.Context, string, string) error
	StoreStatus(context.Context, jobmap.SourceStatus) error
	Statuses(context.Context) ([]jobmap.SourceStatus, error)
	ReserveRequest(context.Context, string, string, int) (bool, error)
	ClaimSearch(context.Context, string, time.Duration) (bool, error)
	ReleaseSearch(context.Context, string) error
	ClaimExternalSearch(context.Context, string, string, time.Duration, time.Duration) (bool, error)
	CompleteExternalSearch(context.Context, string, string) error
	ReleaseExternalSearch(context.Context, string, string) error
	FailExternalSearch(context.Context, string, string, time.Time) error
	Prune(context.Context) error
	FeedProgress(context.Context, string) (jobmap.FeedProgress, error)
	StoreFeedPage(context.Context, string, []jobmap.Job, jobmap.FeedProgress) error
	DelayFeed(context.Context, string, time.Time) error
	MarkJobClosed(context.Context, string, time.Time) error
}

type JobService struct {
	repo               jobStore
	client             *jobmap.Client
	cfg                jobmap.Config
	refreshing         atomic.Bool
	searchGate         chan struct{}
	placeMu            sync.Mutex
	placeCache         map[string]placeCacheEntry
	externalGate       chan struct{}
	externalRefreshing atomic.Bool
	workMu             sync.Mutex
	workCtx            context.Context
	cancelWork         context.CancelFunc
	workWG             sync.WaitGroup
	started            bool
	stopping           bool
	stopped            chan struct{}
}

type placeCacheEntry struct {
	Places []jobmap.Place
	At     time.Time
}

func NewJobService(repo jobStore, cfg jobmap.Config) *JobService {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobService{repo: repo, cfg: cfg, client: jobmap.NewClient(cfg), searchGate: make(chan struct{}, 1), placeCache: map[string]placeCacheEntry{}, externalGate: make(chan struct{}, 1), workCtx: ctx, cancelWork: cancel, stopped: make(chan struct{})}
}

func (s *JobService) refresh(ctx context.Context) {
	if !s.refreshing.CompareAndSwap(false, true) {
		slog.Debug("employer_sync_skipped", "reason", "already_running")
		return
	}
	defer s.refreshing.Store(false)
	claimed, err := s.repo.ClaimSearch(ctx, "company-feed-refresh", s.cfg.RefreshInterval)
	if err != nil {
		logJobError(ctx, "employer_sync_lease_failed", err)
		return
	}
	if !claimed {
		slog.Info("employer_sync_skipped", "reason", "refresh_lease_active", "refresh_interval", s.cfg.RefreshInterval.String())
		return
	}
	started := time.Now()
	slog.Info("employer_sync_started", "sources", len(s.cfg.Companies))
	previous, err := s.repo.Statuses(ctx)
	if err != nil {
		logJobError(ctx, "job_source_status_read_failed", err, "operation", "employer_sync")
	}
	old := map[string]jobmap.SourceStatus{}
	for _, status := range previous {
		old[status.ID] = status
	}
	var wg sync.WaitGroup
	gate := make(chan struct{}, 2)
	failed := atomic.Bool{}
	run := func(id, name, kind string, complete bool, fetch func(context.Context) ([]jobmap.Job, error)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case gate <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-gate }()
			sourceStarted := time.Now()
			slog.Info("employer_source_started", "source", id, "complete_feed", complete)
			status := old[id]
			status.ID, status.Name, status.Kind, status.Status, status.Message = id, name, kind, "refreshing", ""
			s.storeSourceStatus(ctx, status)
			fetchCtx, cancel := context.WithTimeout(ctx, 100*time.Second)
			defer cancel()
			jobs, err := fetch(fetchCtx)
			if err == nil {
				valid := []jobmap.Job{}
				now := time.Now().UTC()
				for _, job := range jobs {
					if j, ok := jobmap.Prepare(job, now); ok {
						valid = append(valid, j)
					}
				}
				err = s.repo.StoreSource(ctx, id, valid, complete)
				if err == nil {
					status.JobCount = len(valid)
					status.RefreshedAt = &now
				}
			}
			status.Status = "ready"
			if err != nil {
				status.Status = "error"
				status.Message = "Could not refresh this source. Recent cached listings may still be available."
				failed.Store(true)
				logJobError(ctx, "employer_source_failed", err, "source", id, "duration_ms", time.Since(sourceStarted).Milliseconds())
			} else {
				slog.Info("employer_source_completed", "source", id, "received", len(jobs), "accepted", status.JobCount, "duration_ms", time.Since(sourceStarted).Milliseconds())
			}
			s.storeSourceStatus(ctx, status)
		}()
	}
	for _, company := range s.cfg.Companies {
		company := company
		kind := "company"
		if company.Provider == "career_page" {
			kind = "career_page"
		}
		run(jobmap.CompanySource(company), company.Name+" careers", kind, company.Provider != "career_page", func(ctx context.Context) ([]jobmap.Job, error) { return s.client.FetchCompany(ctx, company) })
	}
	wg.Wait()
	// Allow another attempt after an outage, without retrying on each user search.
	if failed.Load() || ctx.Err() != nil {
		s.releaseSearch(ctx, "company-feed-refresh", "employers")
	}
	if err := s.repo.Prune(ctx); err != nil {
		logJobError(ctx, "job_cleanup_failed", err)
	}
	slog.Info("employer_sync_finished", "source_errors", failed.Load(), "duration_ms", time.Since(started).Milliseconds())
}

func (s *JobService) sourceStatuses(ctx context.Context) ([]jobmap.SourceStatus, error) {
	saved, err := s.repo.Statuses(ctx)
	if err != nil {
		return nil, err
	}
	byID := map[string]jobmap.SourceStatus{}
	for _, status := range saved {
		byID[status.ID] = status
	}
	defaults := []jobmap.SourceStatus{{ID: "himalayas", Name: "Himalayas", Kind: "aggregator", Status: "refreshing"}, {ID: "jsearch", Name: "JSearch", Kind: "aggregator", Status: "ready"}}
	for _, c := range s.cfg.Companies {
		kind := "company"
		if c.Provider == "career_page" {
			kind = "career_page"
		}
		defaults = append(defaults, jobmap.SourceStatus{ID: jobmap.CompanySource(c), Name: c.Name + " careers", Kind: kind, Status: "refreshing"})
	}
	for i := range defaults {
		if old, ok := byID[defaults[i].ID]; ok {
			defaults[i] = old
		}
		if defaults[i].ID == "jsearch" && s.cfg.JSearchKey == "" {
			defaults[i].Status = "needs_key"
			defaults[i].Message = "Optional broad search is not connected. Company feeds and remote jobs remain available."
		}
	}
	sort.SliceStable(defaults, func(i, j int) bool { return defaults[i].Name < defaults[j].Name })
	return defaults, nil
}

func (s *JobService) Search(ctx context.Context, userID string, search jobmap.Search) (jobmap.SearchResult, error) {
	if userID == "" {
		return jobmap.SearchResult{}, ErrNotFound
	}
	if err := ctx.Err(); err != nil {
		return jobmap.SearchResult{}, err
	}
	queued := false
	if !search.SavedOnly && s.cfg.JSearchKey != "" && search.SourceKind != "company" && search.SourceKind != "career_page" {
		queued = s.queueExternal(ctx, search)
	}
	jobs, err := s.repo.List(ctx, userID, search.SavedOnly)
	if err != nil {
		return jobmap.SearchResult{}, err
	}
	allowed := map[string]bool{"himalayas": true, "jsearch": s.cfg.JSearchKey != ""}
	for _, c := range s.cfg.Companies {
		allowed[jobmap.CompanySource(c)] = true
	}
	visible := []jobmap.Job{}
	for _, job := range jobs {
		if search.SavedOnly || allowed[job.Source] {
			visible = append(visible, job)
		}
	}
	result := jobmap.FilterJobs(visible, search, time.Now().UTC())
	// Details use a separate endpoint, avoiding large descriptions on every list page.
	for i := range result.Jobs {
		runes := []rune(result.Jobs[i].Description)
		if len(runes) > 320 {
			result.Jobs[i].Description = string(runes[:320]) + "…"
		}
	}
	result.Sources, err = s.sourceStatuses(ctx)
	if err != nil {
		return jobmap.SearchResult{}, err
	}
	result.Refreshing = queued || s.refreshing.Load() || s.externalRefreshing.Load()
	return result, nil
}

func (s *JobService) Get(ctx context.Context, userID, id string) (jobmap.Job, error) {
	if userID == "" {
		return jobmap.Job{}, ErrNotFound
	}
	job, err := s.repo.Get(ctx, userID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return job, ErrNotFound
	}
	if err == nil && job.Source == "himalayas" {
		return s.verifyHimalayas(ctx, userID, job)
	}
	return job, err
}
func (s *JobService) Save(ctx context.Context, userID, id string) error {
	if userID == "" {
		return ErrNotFound
	}
	err := s.repo.Save(ctx, userID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
func (s *JobService) Unsave(ctx context.Context, userID, id string) error {
	if userID == "" {
		return ErrNotFound
	}
	return s.repo.Unsave(ctx, userID, id)
}

func (s *JobService) Places(ctx context.Context, query string) ([]jobmap.Place, error) {
	local := jobmap.SearchPlaces(query, 8)
	if s.cfg.GeoapifyKey == "" || len(local) > 0 || len([]rune(query)) < 3 {
		return local, nil
	}
	key := strings.ToLower(strings.TrimSpace(query))
	s.placeMu.Lock()
	cached, ok := s.placeCache[key]
	s.placeMu.Unlock()
	if ok && time.Since(cached.At) < 24*time.Hour {
		return cached.Places, nil
	}
	// Do not consume the geocoder budget for each duplicate in-flight query.
	select {
	case s.searchGate <- struct{}{}:
		defer func() { <-s.searchGate }()
	default:
		return local, nil
	}
	allowed, err := s.repo.ReserveRequest(ctx, "geoapify", time.Now().UTC().Format("2006-01-02"), s.cfg.GeoapifyLimit)
	if err != nil {
		logJobError(ctx, "geocoding_budget_failed", err)
		return nil, err
	}
	if !allowed {
		slog.Warn("geocoding_quota_exhausted", "daily_limit", s.cfg.GeoapifyLimit)
		return local, nil
	}
	places, err := s.client.Geocode(ctx, query)
	if err != nil {
		logJobError(ctx, "geocoding_failed", err)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("location search is unavailable; try a major city: %w", ErrValidation)
	}
	s.placeMu.Lock()
	if len(s.placeCache) >= 256 {
		s.placeCache = map[string]placeCacheEntry{}
	}
	s.placeCache[key] = placeCacheEntry{places, time.Now()}
	s.placeMu.Unlock()
	return places, nil
}

func (s *JobService) Nearest(lat, lng float64) *jobmap.Place { return jobmap.NearestPlace(lat, lng) }
func (s *JobService) Countries() []jobmap.Country            { return jobmap.Countries() }

// Original source URLs are not fetched from user input. Apply remains an
// outbound navigation; description rendering never executes provider HTML.
func (s *JobService) Companies() []jobmap.Company { return s.cfg.Companies }
