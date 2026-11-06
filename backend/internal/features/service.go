package features

import (
	"context"
	"log/slog"
	"maps"
	"sync"
	"time"
)

const featureCacheTTL = 10 * time.Second

type featureStore interface {
	List(context.Context) (map[string]bool, error)
}

// Service shares one bounded snapshot across module gates and the UI.
// Expired enabled values are never used when refreshing the registry fails.
type Service struct {
	definitions []Definition
	store       featureStore
	mu          sync.Mutex
	values      map[Key]bool
	expires     time.Time
	now         func() time.Time
}

func NewService(store featureStore) *Service {
	return &Service{store: store, definitions: Catalog(), now: time.Now}
}

func (s *Service) Snapshot(ctx context.Context) map[Key]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.now().Before(s.expires) {
		return maps.Clone(s.values)
	}
	values := make(map[Key]bool)
	for _, definition := range s.definitions {
		values[definition.Key] = false
	}
	work, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	stored, err := s.store.List(work)
	ttl := featureCacheTTL
	if err != nil {
		// Retry soon, without flooding the database or exposing stale
		ttl = time.Second
		slog.WarnContext(ctx, "feature_registry_unavailable")
	} else {
		for key := range values {
			values[key] = stored[string(key)]
		}
	}
	s.values = values
	s.expires = s.now().Add(ttl)
	return maps.Clone(values)
}

func (s *Service) Enabled(ctx context.Context, key Key) bool {
	if !known(key, s.definitions) {
		return false
	}
	return s.Snapshot(ctx)[key]
}
