package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"resume-builder/backend/internal/jobmap"
)

type feedTestStore struct {
	jobStore // unused operations deliberately fail if unexpectedly called
	progress jobmap.FeedProgress
	jobs     map[string]jobmap.Job
	status   jobmap.SourceStatus
	leases   map[string]time.Time
	requests int
	quota    bool
	failPage bool
}

func newFeedTestStore() *feedTestStore {
	return &feedTestStore{jobs: map[string]jobmap.Job{}, leases: map[string]time.Time{}, quota: true}
}
func (s *feedTestStore) FeedProgress(context.Context, string) (jobmap.FeedProgress, error) {
	return s.progress, nil
}
func (s *feedTestStore) ClaimSearch(_ context.Context, key string, ttl time.Duration) (bool, error) {
	if key == "himalayas-request-slot" {
		return true, nil
	}
	if until := s.leases[key]; until.After(time.Now()) {
		return false, nil
	}
	s.leases[key] = time.Now().Add(ttl)
	return true, nil
}
func (s *feedTestStore) ReleaseSearch(_ context.Context, key string) error {
	delete(s.leases, key)
	return nil
}
func (s *feedTestStore) ReserveRequest(_ context.Context, provider, period string, limit int) (bool, error) {
	if provider != "himalayas" || period != time.Now().UTC().Format("2006-01-02") || limit != 10000 {
		return false, errors.New("incorrect provider budget")
	}
	s.requests++
	return s.quota, nil
}
func (s *feedTestStore) StoreStatus(_ context.Context, status jobmap.SourceStatus) error {
	s.status = status
	return nil
}
func (s *feedTestStore) DelayFeed(_ context.Context, _ string, until time.Time) error {
	s.progress.RetryAt = &until
	return nil
}
func (s *feedTestStore) StoreFeedPage(ctx context.Context, source string, jobs []jobmap.Job, p jobmap.FeedProgress) error {
	if s.failPage {
		return errors.New("transaction failed")
	}
	if err := s.StoreSource(ctx, source, jobs, false); err != nil {
		return err
	}
	p.RetryAt = s.progress.RetryAt
	s.progress = p
	return nil
}
func (s *feedTestStore) StoreSource(_ context.Context, _ string, jobs []jobmap.Job, complete bool) error {
	if complete {
		return errors.New("cursor feed must never close unseen jobs")
	}
	for _, j := range jobs {
		old := s.jobs[j.ID]
		j.Saved = old.Saved
		if j.CheckedAt == nil {
			j.Closed, j.CheckedAt = old.Closed, old.CheckedAt
		}
		s.jobs[j.ID] = j
	}
	return nil
}
func (s *feedTestStore) Get(_ context.Context, _ string, id string) (jobmap.Job, error) {
	return s.jobs[id], nil
}
func (s *feedTestStore) MarkJobClosed(_ context.Context, id string, at time.Time) error {
	j := s.jobs[id]
	j.Closed, j.CheckedAt = true, &at
	s.jobs[id] = j
	return nil
}

type feedTestTransport func(*http.Request) (*http.Response, error)

func (f feedTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func feedTestResponse(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}

const remoteFeedJob = `{"guid":"https://himalayas.app/companies/acme/jobs/a","title":"Software Engineer","companyName":"Acme","applicationLink":"https://example.com/apply","locationRestrictions":[],"pubDate":1700000000}`

func TestHimalayasSyncResumesAfterFailureAndRunsDaily(t *testing.T) {
	store := newFeedTestStore()
	store.jobs["unseen"] = jobmap.Job{ID: "unseen"}
	svc := NewJobService(store, jobmap.Config{})
	calls := 0
	svc.client.HTTP.Transport = feedTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			if r.URL.Query().Get("cursor") != "" {
				t.Fatal("first page had cursor")
			}
			return feedTestResponse(200, `{"jobs":[`+remoteFeedJob+`],"nextCursor":"resume-here"}`), nil
		}
		return feedTestResponse(503, `{}`), nil
	})
	svc.refreshHimalayas(context.Background())
	if store.progress.Cursor != "resume-here" || store.progress.CompletedAt != nil || len(store.jobs) != 2 || store.status.Status != "error" {
		t.Fatalf("lost partial progress: %+v %+v", store.progress, store.status)
	}
	// A new service instance must use the durable cursor, not start at page one.
	svc = NewJobService(store, jobmap.Config{})
	svc.client.HTTP.Transport = feedTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Query().Get("cursor") != "resume-here" {
			t.Fatal("did not resume", r.URL)
		}
		return feedTestResponse(200, `{"jobs":[]}`), nil
	})
	svc.refreshHimalayas(context.Background())
	if store.progress.CompletedAt == nil || store.progress.Cursor != "" || store.status.Status != "ready" || store.jobs["unseen"].Closed {
		t.Fatalf("invalid completion: %+v", store.progress)
	}
	svc.refreshHimalayas(context.Background())
	if calls != 3 {
		t.Fatal("completed feed polled before daily interval", calls)
	}
}

func TestHimalayasFailedPageDoesNotAdvanceCheckpoint(t *testing.T) {
	store := newFeedTestStore()
	store.failPage = true
	svc := NewJobService(store, jobmap.Config{})
	svc.client.HTTP.Transport = feedTestTransport(func(*http.Request) (*http.Response, error) {
		return feedTestResponse(200, `{"jobs":[`+remoteFeedJob+`],"nextCursor":"next"}`), nil
	})
	svc.refreshHimalayas(context.Background())
	if store.progress.Cursor != "" || len(store.jobs) != 0 || store.status.Status != "error" {
		t.Fatal("uncommitted page advanced cursor")
	}
}

func TestHimalayasDailyBudgetStopsNetworkRequests(t *testing.T) {
	store := newFeedTestStore()
	store.quota = false
	svc := NewJobService(store, jobmap.Config{})
	svc.client.HTTP.Transport = feedTestTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("exhausted budget made a request")
		return nil, nil
	})
	svc.refreshHimalayas(context.Background())
	if store.progress.RetryAt == nil || !store.progress.RetryAt.After(time.Now()) || store.status.Status != "error" {
		t.Fatal("budget exhaustion didn't defer collection")
	}
}

func TestHimalayasAvailabilityChecksPreserveBookmarksAndHandleOutages(t *testing.T) {
	for _, code := range []int{200, 404, 410, 429, 503} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			store := newFeedTestStore()
			job, ok := jobmap.Prepare(jobmap.Job{ProviderID: "https://himalayas.app/companies/acme/jobs/a", Source: "himalayas", Company: "Acme", Title: "Software Engineer", ApplyURL: "https://example.com/apply"}, time.Now())
			if !ok {
				t.Fatal("invalid fixture")
			}
			job.Saved = true
			store.jobs[job.ID] = job
			svc := NewJobService(store, jobmap.Config{})
			calls := 0
			svc.client.HTTP.Transport = feedTestTransport(func(*http.Request) (*http.Response, error) {
				calls++
				return feedTestResponse(code, `{"job":`+remoteFeedJob+`}`), nil
			})
			got, err := svc.Get(context.Background(), "user", job.ID)
			failed := code != 200 && code != 410
			if err != nil || !got.Saved || got.Closed != (code == 410) || got.CheckFailed != failed || (got.CheckedAt != nil) == failed {
				t.Fatalf("status %d: %+v err %v", code, got, err)
			}
			if code == 429 && store.progress.RetryAt == nil {
				t.Fatal("429 did not defer provider requests")
			}
			if _, err := svc.Get(context.Background(), "user", job.ID); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatal("repeated opening bypassed check cache")
			}
		})
	}
}
