package service

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"resume-builder/backend/internal/jobmap"
)

type externalTestStore struct {
	jobStore
	mu          sync.Mutex
	leases      map[string]string
	fresh       map[string]bool
	retryAt     map[string]time.Time
	jobs        []jobmap.Job
	statuses    map[string]jobmap.SourceStatus
	requests    int
	quota       bool
	budgetError error
	completed   chan struct{}
	failed      chan struct{}
}

func newExternalTestStore() *externalTestStore {
	return &externalTestStore{leases: map[string]string{}, fresh: map[string]bool{}, retryAt: map[string]time.Time{}, statuses: map[string]jobmap.SourceStatus{}, quota: true, completed: make(chan struct{}, 4), failed: make(chan struct{}, 4)}
}
func (s *externalTestStore) ClaimExternalSearch(ctx context.Context, key, token string, _, _ time.Duration) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.fresh[key] || s.leases[key] != "" || s.retryAt[key].After(time.Now()) {
		return false, nil
	}
	s.leases[key] = token
	return true, nil
}
func (s *externalTestStore) ReleaseExternalSearch(ctx context.Context, key, token string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.leases[key] == token {
		delete(s.leases, key)
	}
	return nil
}

func (s *externalTestStore) FailExternalSearch(ctx context.Context, key, token string, until time.Time) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.leases[key] == token {
		delete(s.leases, key)
		s.retryAt[key] = until
		s.failed <- struct{}{}
	}
	return nil
}
func (s *externalTestStore) CompleteExternalSearch(ctx context.Context, key, token string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.leases[key] != token {
		return errors.New("lost lease")
	}
	s.fresh[key] = true
	delete(s.leases, key)
	s.completed <- struct{}{}
	return nil
}
func (s *externalTestStore) ReserveRequest(ctx context.Context, _, _ string, _ int) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.budgetError != nil {
		return false, s.budgetError
	}
	if s.quota {
		s.requests++
	}
	return s.quota, nil
}
func (s *externalTestStore) StoreSource(ctx context.Context, _ string, jobs []jobmap.Job, _ bool) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append([]jobmap.Job{}, jobs...)
	return nil
}
func (s *externalTestStore) StoreStatus(ctx context.Context, status jobmap.SourceStatus) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses[status.ID] = status
	return nil
}
func (s *externalTestStore) Statuses(ctx context.Context) ([]jobmap.SourceStatus, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []jobmap.SourceStatus{}
	for _, status := range s.statuses {
		out = append(out, status)
	}
	return out, nil
}
func (s *externalTestStore) List(ctx context.Context, _ string, _ bool) ([]jobmap.Job, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]jobmap.Job{}, s.jobs...), nil
}
func stopTestJobs(t *testing.T, svc *JobService) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := svc.Stop(ctx); err != nil {
		t.Error(err)
	}
}
func waitTestSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal("background operation did not finish")
	}
}

const externalJobResponse = `{"status":"OK","data":{"jobs":[{"job_id":"one","job_title":"Software Engineer","employer_name":"Acme","job_apply_link":"https://example.com/apply","job_is_remote":true,"job_location":"Worldwide"}]}}`

func TestSearchSurvivesBrowserCancellationAndCachesStoredResults(t *testing.T) {
	store := newExternalTestStore()
	svc := NewJobService(store, jobmap.Config{JSearchKey: "test-key", JSearchLimit: 200})
	t.Cleanup(func() { stopTestJobs(t, svc) })
	started := make(chan struct{})
	respond := make(chan struct{})
	var providerCtx context.Context
	svc.client.HTTP.Transport = feedTestTransport(func(r *http.Request) (*http.Response, error) {
		providerCtx = r.Context()
		close(started)
		select {
		case <-respond:
			return feedTestResponse(200, externalJobResponse), nil
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	})
	request, cancelRequest := context.WithCancel(context.Background())
	defer cancelRequest()
	result, err := svc.Search(request, "user", jobmap.Search{Query: "software", Country: "IN"})
	if err != nil || !result.Refreshing || result.Total != 0 {
		t.Fatalf("initial database-backed result: %+v %v", result, err)
	}
	waitTestSignal(t, started)
	cancelRequest()
	if providerCtx.Err() != nil {
		t.Fatal("browser cancellation aborted provider import")
	}
	close(respond)
	waitTestSignal(t, store.completed)
	result, err = svc.Search(context.Background(), "user", jobmap.Search{Query: "software", Country: "IN"})
	if err != nil || result.Total != 1 {
		t.Fatal("completed background results missing", result, err)
	}
	stopTestJobs(t, svc)
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.requests != 1 || len(store.leases) != 0 || len(store.fresh) != 1 {
		t.Fatal("duplicate paid request or leaked lock", store.requests, store.leases, store.fresh)
	}
}

func TestShutdownReleasesSearchUsingLiveCleanupContext(t *testing.T) {
	store := newExternalTestStore()
	svc := NewJobService(store, jobmap.Config{JSearchKey: "test-key", JSearchLimit: 200})
	t.Cleanup(func() { stopTestJobs(t, svc) })
	started := make(chan struct{})
	svc.client.HTTP.Transport = feedTestTransport(func(r *http.Request) (*http.Response, error) {
		close(started)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	if !svc.queueExternal(context.Background(), jobmap.Search{Country: "IN"}) {
		t.Fatal("search not queued")
	}
	waitTestSignal(t, started)
	stopTestJobs(t, svc)
	store.mu.Lock()
	if len(store.leases) != 0 || len(store.fresh) != 0 || store.statuses["jsearch"].Status != "error" {
		t.Error("shutdown poisoned cache or lost cleanup", store.leases, store.fresh, store.statuses)
	}
	store.mu.Unlock()
	if svc.queueExternal(context.Background(), jobmap.Search{Country: "US"}) {
		t.Fatal("stopped service launched work")
	}
	restarted := NewJobService(store, jobmap.Config{JSearchKey: "test-key", JSearchLimit: 200})
	t.Cleanup(func() { stopTestJobs(t, restarted) })
	restarted.client.HTTP.Transport = feedTestTransport(func(*http.Request) (*http.Response, error) { return feedTestResponse(200, externalJobResponse), nil })
	if !restarted.queueExternal(context.Background(), jobmap.Search{Country: "IN"}) {
		t.Fatal("interrupted query could not retry after restart")
	}
	waitTestSignal(t, store.completed)
}

func TestExternalQuotaAndBudgetFailureReleaseTheirLocks(t *testing.T) {
	for _, budgetFailure := range []bool{false, true} {
		store := newExternalTestStore()
		store.quota = false
		if budgetFailure {
			store.budgetError = errors.New("database unavailable")
		}
		svc := NewJobService(store, jobmap.Config{JSearchKey: "test-key", JSearchLimit: 200})
		key, token := "query", "owner"
		store.leases[key] = token
		svc.client.HTTP.Transport = feedTestTransport(func(*http.Request) (*http.Response, error) {
			t.Error("quota-denied operation called provider")
			return nil, errors.New("unexpected request")
		})
		svc.searchExternal(context.Background(), "software in India", "IN", key, token)
		want := "quota_exhausted"
		if budgetFailure {
			want = "error"
		}
		if len(store.leases) != 0 || len(store.fresh) != 0 || store.statuses["jsearch"].Status != want {
			t.Fatal("incorrect budget handling", store.statuses, store.leases)
		}
		stopTestJobs(t, svc)
	}
}

func TestProviderFailureDoesNotConsumeQuotaOnEveryPoll(t *testing.T) {
	store := newExternalTestStore()
	svc := NewJobService(store, jobmap.Config{JSearchKey: "test-key", JSearchLimit: 200})
	t.Cleanup(func() { stopTestJobs(t, svc) })
	svc.client.HTTP.Transport = feedTestTransport(func(*http.Request) (*http.Response, error) { return feedTestResponse(503, `{}`), nil })
	if !svc.queueExternal(context.Background(), jobmap.Search{Country: "IN"}) {
		t.Fatal("initial query not scheduled")
	}
	waitTestSignal(t, store.failed)
	deadline := time.After(5 * time.Second)
	for svc.externalRefreshing.Load() {
		select {
		case <-deadline:
			t.Fatal("failed provider worker stuck")
		case <-time.After(time.Millisecond):
		}
	}
	if svc.queueExternal(context.Background(), jobmap.Search{Country: "IN"}) {
		t.Fatal("UI poll bypassed failed-query backoff")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.requests != 1 || len(store.fresh) != 0 || len(store.retryAt) != 1 {
		t.Fatal("failure poisoned success cache or consumed extra quota")
	}
}
