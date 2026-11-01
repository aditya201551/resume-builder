package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"resume-builder/backend/internal/jobmap"
)

// Shared lease/budget fixture for on-demand discovery and job-assistant tests.
type discoveryLeaseStore struct {
	mu       sync.Mutex
	leases   map[string]string
	retryAt  map[string]time.Time
	requests int
	quota    bool
}

func newDiscoveryLeaseStore() *discoveryLeaseStore {
	return &discoveryLeaseStore{leases: map[string]string{}, retryAt: map[string]time.Time{}, quota: true}
}
func (s *discoveryLeaseStore) ClaimExternalSearch(ctx context.Context, key, token string, _, _ time.Duration) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.leases[key] != "" || s.retryAt[key].After(time.Now()) {
		return false, nil
	}
	s.leases[key] = token
	return true, nil
}
func (s *discoveryLeaseStore) ReleaseExternalSearch(ctx context.Context, key, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.leases[key] == token {
		delete(s.leases, key)
	}
	return nil
}
func (s *discoveryLeaseStore) FailExternalSearch(ctx context.Context, key, token string, until time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.leases[key] == token {
		delete(s.leases, key)
		s.retryAt[key] = until
	}
	return nil
}
func (s *discoveryLeaseStore) CompleteExternalSearch(ctx context.Context, key, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.leases[key] != token {
		return errors.New("lost lease")
	}
	delete(s.leases, key)
	return nil
}
func (s *discoveryLeaseStore) ReserveRequest(ctx context.Context, _, _ string, _ int) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quota {
		s.requests++
	}
	return s.quota, nil
}

type himalayasTestStore struct {
	jobStore
	jobs     map[string]jobmap.Job
	leases   map[string]time.Time
	retryAt  *time.Time
	requests int
	quota    bool
}

func newHimalayasTestStore() *himalayasTestStore {
	return &himalayasTestStore{jobs: map[string]jobmap.Job{}, leases: map[string]time.Time{}, quota: true}
}
func (s *himalayasTestStore) ProviderRetryAt(context.Context, string) (*time.Time, error) {
	return s.retryAt, nil
}
func (s *himalayasTestStore) ClaimSearch(_ context.Context, key string, ttl time.Duration) (bool, error) {
	if key == "himalayas-request-slot" {
		return true, nil
	}
	if s.leases[key].After(time.Now()) {
		return false, nil
	}
	s.leases[key] = time.Now().Add(ttl)
	return true, nil
}
func (s *himalayasTestStore) ReleaseSearch(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	delete(s.leases, key)
	return nil
}
func (s *himalayasTestStore) ReserveRequest(_ context.Context, provider, period string, limit int) (bool, error) {
	if provider != "himalayas" || period != time.Now().UTC().Format("2006-01-02") || limit != 10000 {
		return false, errors.New("incorrect provider budget")
	}
	s.requests++
	return s.quota, nil
}
func (s *himalayasTestStore) DelayFeed(_ context.Context, _ string, until time.Time) error {
	s.retryAt = &until
	return nil
}
func (s *himalayasTestStore) StoreSource(_ context.Context, _ string, jobs []jobmap.Job) error {
	for _, j := range jobs {
		j.Saved = s.jobs[j.ID].Saved
		s.jobs[j.ID] = j
	}
	return nil
}
func (s *himalayasTestStore) Get(_ context.Context, _ string, id string) (jobmap.Job, error) {
	return s.jobs[id], nil
}
func (s *himalayasTestStore) MarkJobClosed(_ context.Context, id string, at time.Time) error {
	j := s.jobs[id]
	j.Closed = true
	j.CheckedAt = &at
	s.jobs[id] = j
	return nil
}

type jobTestTransport func(*http.Request) (*http.Response, error)

func (f jobTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func jobTestResponse(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}

const himalayasJobJSON = `{"guid":"https://himalayas.app/companies/acme/jobs/a","title":"Software Engineer","companyName":"Acme","applicationLink":"https://example.com/apply","locationRestrictions":[],"pubDate":1700000000}`
const jsearchJobResponse = `{"status":"OK","data":{"jobs":[{"job_id":"one","job_title":"Software Engineer","employer_name":"Acme","job_apply_link":"https://example.com/apply","job_is_remote":true,"job_location":"Worldwide"}]}}`
