package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"resume-builder/backend/internal/jobmap"
	"resume-builder/backend/internal/jobsearch"
	"strings"
	"testing"
)

type discoveryTestStore struct {
	*discoveryLeaseStore
	pages map[string]jobsearch.Page
}

func (s *discoveryTestStore) DiscoveryPage(_ context.Context, userID, key string) (jobsearch.Page, bool, error) {
	p, ok := s.pages[key]
	p.Jobs = append([]jobmap.Job{}, p.Jobs...)
	for i := range p.Jobs {
		p.Jobs[i].Saved = userID == "bookmarked-user"
	}
	return p, ok, nil
}
func (s *discoveryTestStore) StoreDiscoveryPage(_ context.Context, key string, p jobsearch.Page) error {
	s.pages[key] = p
	return nil
}
func (s *discoveryTestStore) SavedDiscoveryPage(_ context.Context, userID string, _ int) ([]jobmap.Job, bool, error) {
	if userID == "bookmarked-user" {
		return []jobmap.Job{{ID: "saved", Saved: true, Workplace: "remote", Remote: jobmap.RemotePolicy{Scope: "worldwide"}}}, false, nil
	}
	return []jobmap.Job{}, false, nil
}

func TestDiscoveryFetchesRequestedHimalayasPagesAndSharesCache(t *testing.T) {
	store := &discoveryTestStore{discoveryLeaseStore: newDiscoveryLeaseStore(), pages: map[string]jobsearch.Page{}}
	worker := NewJobService(newHimalayasTestStore(), jobmap.Config{})
	calls := 0
	worker.client.HTTP.Transport = jobTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/jobs/api/search" || r.URL.Query().Get("country") != "IN" {
			t.Errorf("unfiltered or wrongly scoped request: %s", r.URL)
		}
		if r.URL.Query().Get("page") == "2" {
			return jobTestResponse(200, `{"jobs":[]}`), nil
		}
		return jobTestResponse(200, `{"jobs":[`+himalayasJobJSON+`]}`), nil // short page still has more
	})
	svc := NewJobDiscoveryService(store, worker)
	intent := jobsearch.Intent{Version: 1, Query: "engineer", Branches: []jobsearch.Branch{{Workplace: "remote", Country: "IN"}}}
	first, err := svc.Discover(context.Background(), "user", jobsearch.Request{Intent: intent})
	if err != nil || len(first.Jobs) != 1 || first.NextToken == "" || first.Retry || calls != 1 {
		t.Fatal(first, err, calls)
	}
	// Cache is public, bookmark state is not. Local filters reuse a provider page.
	intent.IncludeUnknown = true
	second, err := svc.Discover(context.Background(), "bookmarked-user", jobsearch.Request{Intent: intent})
	if err != nil || len(second.Jobs) != 1 || !second.Jobs[0].Saved || calls != 1 {
		t.Fatal("cache or user isolation failed", second, err, calls)
	}
	end, err := svc.Discover(context.Background(), "user", jobsearch.Request{Intent: jobsearch.Intent{Version: 1, Query: "engineer", Branches: []jobsearch.Branch{{Workplace: "remote", Country: "IN"}}}, NextToken: first.NextToken})
	if err != nil || len(end.Jobs) != 0 || end.NextToken != "" || calls != 2 {
		t.Fatal(end, err, calls)
	}
	intent.Query = "different"
	if _, err := svc.Discover(context.Background(), "user", jobsearch.Request{Intent: intent, NextToken: first.NextToken}); err == nil {
		t.Fatal("old continuation accepted for changed criteria")
	}
}

func TestDiscoveryJSearchCursorAndNoDefaultCountry(t *testing.T) {
	store := &discoveryTestStore{discoveryLeaseStore: newDiscoveryLeaseStore(), pages: map[string]jobsearch.Page{}}
	worker := NewJobService(newHimalayasTestStore(), jobmap.Config{JSearchKey: "key", JSearchLimit: 10})
	calls := 0
	worker.client.HTTP.Transport = jobTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Query().Get("country") != "de" {
			t.Error("country lost")
		}
		if calls == 2 && r.URL.Query().Get("cursor") != "opaque-next" {
			t.Error("cursor lost")
		}
		body := strings.Replace(jsearchJobResponse, `"jobs":`, `"cursor":"opaque-next","jobs":`, 1)
		if calls == 2 {
			body = `{"status":"OK","data":{"jobs":[]}}`
		}
		return jobTestResponse(200, body), nil
	})
	svc := NewJobDiscoveryService(store, worker)
	intent := jobsearch.Intent{Version: 1, Branches: []jobsearch.Branch{{Workplace: "onsite", Country: "DE"}}}
	first, err := svc.Discover(context.Background(), "user", jobsearch.Request{Intent: intent})
	if err != nil || first.NextToken == "" || calls != 1 {
		t.Fatal(first, err, calls)
	}
	_, err = svc.Discover(context.Background(), "user", jobsearch.Request{Intent: intent, NextToken: first.NextToken})
	if err != nil || calls != 2 {
		t.Fatal(err, calls)
	}
	intent.Branches[0].Country = ""
	r, err := svc.Discover(context.Background(), "user", jobsearch.Request{Intent: intent})
	if err != nil || calls != 2 || r.Sources[0].Status != "needs_location" {
		t.Fatal(r, err, calls)
	}
}

func TestDiscoverySavedDoesNotCallProviders(t *testing.T) {
	store := &discoveryTestStore{discoveryLeaseStore: newDiscoveryLeaseStore(), pages: map[string]jobsearch.Page{}}
	svc := NewJobDiscoveryService(store, NewJobService(nil, jobmap.Config{}))
	intent := jobsearch.Intent{Version: 1, Branches: []jobsearch.Branch{{Workplace: "remote"}}}
	r, err := svc.Discover(context.Background(), "bookmarked-user", jobsearch.Request{Intent: intent, SavedOnly: true})
	if err != nil || len(r.Jobs) != 1 {
		t.Fatal(r, err)
	}
	r, err = svc.Discover(context.Background(), "other-user", jobsearch.Request{Intent: intent, SavedOnly: true})
	if err != nil || len(r.Jobs) != 0 {
		t.Fatal(r, err)
	}
}

func TestDiscoveryFailureRetainsCursorForRetry(t *testing.T) {
	store := &discoveryTestStore{discoveryLeaseStore: newDiscoveryLeaseStore(), pages: map[string]jobsearch.Page{}}
	worker := NewJobService(newHimalayasTestStore(), jobmap.Config{})
	worker.client.HTTP.Transport = jobTestTransport(func(*http.Request) (*http.Response, error) { return jobTestResponse(503, `{}`), nil })
	svc := NewJobDiscoveryService(store, worker)
	intent := jobsearch.Intent{Version: 1, Branches: []jobsearch.Branch{{Workplace: "remote"}}}
	r, err := svc.Discover(context.Background(), "user", jobsearch.Request{Intent: intent})
	if err != nil || !r.Retry || r.NextToken == "" {
		t.Fatal(r, err)
	}
	data, _ := base64.RawURLEncoding.DecodeString(r.NextToken)
	var state discoveryCursor
	_ = json.Unmarshal(data, &state)
	if state.Positions[0].Page != 1 || state.Positions[0].Done {
		t.Fatal("failed request skipped a page", state)
	}
}
