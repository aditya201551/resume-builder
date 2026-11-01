package service

import (
	"context"
	"net/http"
	"resume-builder/backend/internal/jobmap"
	"testing"
	"time"
)

func TestHimalayasAvailabilityChecksPreserveBookmarksAndHandleOutages(t *testing.T) {
	for _, code := range []int{200, 404, 410, 429, 503} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			store := newHimalayasTestStore()
			job, ok := jobmap.Prepare(jobmap.Job{ProviderID: "https://himalayas.app/companies/acme/jobs/a", Source: "himalayas", Company: "Acme", Title: "Software Engineer", ApplyURL: "https://example.com/apply"}, time.Now())
			if !ok {
				t.Fatal("invalid fixture")
			}
			job.Saved = true
			store.jobs[job.ID] = job
			svc := NewJobService(store, jobmap.Config{})
			calls := 0
			svc.client.HTTP.Transport = jobTestTransport(func(*http.Request) (*http.Response, error) {
				calls++
				return jobTestResponse(code, `{"job":`+himalayasJobJSON+`}`), nil
			})
			got, err := svc.Get(context.Background(), "user", job.ID)
			failed := code != 200 && code != 410
			if err != nil || !got.Saved || got.Closed != (code == 410) || got.CheckFailed != failed || (got.CheckedAt != nil) == failed {
				t.Fatalf("status %d: %+v err %v", code, got, err)
			}
			if code == 429 && store.retryAt == nil {
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

func TestHimalayasDailyBudgetStopsSearchRequests(t *testing.T) {
	store := newHimalayasTestStore()
	store.quota = false
	svc := NewJobService(store, jobmap.Config{})
	if err := svc.reserveHimalayasRequest(context.Background()); err == nil {
		t.Fatal("exhausted daily budget was accepted")
	}
	if store.retryAt == nil || !store.retryAt.After(time.Now()) {
		t.Fatal("exhausted budget did not establish retry time")
	}
}
