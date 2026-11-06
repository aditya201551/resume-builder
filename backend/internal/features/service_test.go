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

const testFeature Key = "test.enabled"

func newFeatureTestService(store featureStore) *Service {
	service := NewService(store)
	service.definitions = []Definition{{Key: testFeature, Description: "Test feature"}}
	return service
}

func TestFeatureSnapshotRefreshesAndFailsClosed(t *testing.T) {
	store := &featureTestStore{values: map[string]bool{"test.enabled": true, "unknown": true}}
	svc := newFeatureTestService(store)
	now := time.Now()
	svc.now = func() time.Time { return now }
	ctx := context.Background()
	if !svc.Enabled(ctx, testFeature) || svc.Enabled(ctx, "unknown") {
		t.Fatal("only a known explicitly enabled flag should be available")
	}
	// Callers cannot mutate the shared snapshot.
	snapshot := svc.Snapshot(ctx)
	snapshot[testFeature] = false
	if !svc.Enabled(ctx, testFeature) || store.reads != 1 {
		t.Fatal("cache was mutated or not reused")
	}
	store.values["test.enabled"] = false
	now = now.Add(featureCacheTTL)
	if svc.Enabled(ctx, testFeature) {
		t.Fatal("disabled flag did not refresh")
	}
	store.values["test.enabled"] = true
	now = now.Add(featureCacheTTL)
	if !svc.Enabled(ctx, testFeature) {
		t.Fatal("live re-enable did not refresh")
	}
	store.err = errors.New("database unavailable")
	now = now.Add(featureCacheTTL)
	if svc.Enabled(ctx, testFeature) {
		t.Fatal("expired enabled flag survived a database failure")
	}
	store.err = nil
	delete(store.values, "test.enabled")
	now = now.Add(time.Second)
	if svc.Enabled(ctx, testFeature) {
		t.Fatal("missing flag must default off")
	}
}

func TestFeatureUpdatesValidateAndInvalidateCache(t *testing.T) {
	store := &featureTestStore{values: map[string]bool{"test.enabled": false}}
	svc := newFeatureTestService(store)
	ctx := context.Background()
	_ = svc.Snapshot(ctx)
	for _, test := range []struct {
		key   Key
		actor string
	}{{"invented", "owner"}, {testFeature, " "}} {
		if !errors.Is(svc.Set(ctx, test.key, true, test.actor), ErrValidation) {
			t.Fatal("invalid administrative update accepted")
		}
	}
	if err := svc.Set(ctx, testFeature, true, " owner "); err != nil {
		t.Fatal(err)
	}
	if !svc.Enabled(ctx, testFeature) || store.actor != "owner" {
		t.Fatal("update was not visible or attributable")
	}
}

func TestConcurrentFeatureReadsShareOneRefresh(t *testing.T) {
	store := &featureTestStore{values: map[string]bool{"test.enabled": true}}
	svc := newFeatureTestService(store)
	var group sync.WaitGroup
	for range 30 {
		group.Go(func() {
			if !svc.Enabled(context.Background(), testFeature) {
				t.Error("enabled flag lost")
			}
		})
	}
	group.Wait()
	if store.reads != 1 {
		t.Fatalf("expected one refresh, got %d", store.reads)
	}
}

func TestEmptyCatalogIgnoresRetiredDatabaseFlags(t *testing.T) {
	store := &featureTestStore{values: map[string]bool{"retired.enabled": true}}
	service := NewService(store)
	if len(Catalog()) != 0 || len(service.Snapshot(context.Background())) != 0 || service.Enabled(context.Background(), "retired.enabled") {
		t.Fatal("unregistered database flags must not be exposed")
	}
	if !errors.Is(service.Set(context.Background(), "retired.enabled", true, "owner"), ErrValidation) {
		t.Fatal("retired flags must not accept administrative updates")
	}
}
