package features

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type featureTestStore struct {
	mu     sync.Mutex
	values map[string]bool
	err    error
	reads  int
	actor  string
}

func (s *featureTestStore) List(context.Context) (map[string]bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads++
	return s.values, s.err
}

func (s *featureTestStore) Set(_ context.Context, key string, enabled bool, actor string) error {
	s.values[key] = enabled
	s.actor = actor
	return s.err
}

func TestFeatureSnapshotRefreshesAndFailsClosed(t *testing.T) {
	store := &featureTestStore{values: map[string]bool{"jobs.enabled": true, "unknown": true}}
	svc := NewService(store)
	now := time.Now()
	svc.now = func() time.Time { return now }
	ctx := context.Background()
	if !svc.Enabled(ctx, Jobs) || svc.Enabled(ctx, "unknown") {
		t.Fatal("only a known explicitly enabled flag should be available")
	}
	// Callers cannot mutate the shared snapshot.
	snapshot := svc.Snapshot(ctx)
	snapshot[Jobs] = false
	if !svc.Enabled(ctx, Jobs) || store.reads != 1 {
		t.Fatal("cache was mutated or not reused")
	}
	store.values["jobs.enabled"] = false
	now = now.Add(featureCacheTTL)
	if svc.Enabled(ctx, Jobs) {
		t.Fatal("disabled flag did not refresh")
	}
	store.values["jobs.enabled"] = true
	now = now.Add(featureCacheTTL)
	if !svc.Enabled(ctx, Jobs) {
		t.Fatal("live re-enable did not refresh")
	}
	store.err = errors.New("database unavailable")
	now = now.Add(featureCacheTTL)
	if svc.Enabled(ctx, Jobs) {
		t.Fatal("expired enabled flag survived a database failure")
	}
	store.err = nil
	delete(store.values, "jobs.enabled")
	now = now.Add(time.Second)
	if svc.Enabled(ctx, Jobs) {
		t.Fatal("missing flag must default off")
	}
}

func TestFeatureUpdatesValidateAndInvalidateCache(t *testing.T) {
	store := &featureTestStore{values: map[string]bool{"jobs.enabled": false}}
	svc := NewService(store)
	ctx := context.Background()
	_ = svc.Snapshot(ctx)
	for _, test := range []struct {
		key   Key
		actor string
	}{{"invented", "owner"}, {Jobs, " "}} {
		if !errors.Is(svc.Set(ctx, test.key, true, test.actor), ErrValidation) {
			t.Fatal("invalid administrative update accepted")
		}
	}
	if err := svc.Set(ctx, Jobs, true, " owner "); err != nil {
		t.Fatal(err)
	}
	if !svc.Enabled(ctx, Jobs) || store.actor != "owner" {
		t.Fatal("update was not visible or attributable")
	}
}

func TestConcurrentFeatureReadsShareOneRefresh(t *testing.T) {
	store := &featureTestStore{values: map[string]bool{"jobs.enabled": true}}
	svc := NewService(store)
	var group sync.WaitGroup
	for range 30 {
		group.Go(func() {
			if !svc.Enabled(context.Background(), Jobs) {
				t.Error("enabled flag lost")
			}
		})
	}
	group.Wait()
	if store.reads != 1 {
		t.Fatalf("expected one refresh, got %d", store.reads)
	}
}
