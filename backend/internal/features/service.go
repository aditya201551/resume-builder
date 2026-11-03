package features

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"sync"
	"time"
)

var ErrValidation = errors.New("feature validation error")

const featureCacheTTL = 10 * time.Second

type featureStore interface {
	List(context.Context) (map[string]bool, error)
	Set(context.Context, string, bool, string) error
}

// Service shares one bounded snapshot across module gates and the UI.
// Expired enabled values are never used when refreshing the registry fails.
type Service struct {
	store   featureStore
	mu      sync.Mutex
	values  map[Key]bool
	expires time.Time
	now     func() time.Time
}

func NewService(store featureStore) *Service {
	return &Service{store: store, now: time.Now}
}

func (s *Service) Snapshot(ctx context.Context) map[Key]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.now().Before(s.expires) {
		return maps.Clone(s.values)
	}
	values := make(map[Key]bool)
	for _, definition := range Catalog() {
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
	if !Known(key) {
		return false
	}
	return s.Snapshot(ctx)[key]
}

func (s *Service) Set(ctx context.Context, key Key, enabled bool, actor string) error {
	actor = strings.TrimSpace(actor)
	if !Known(key) || actor == "" || len(actor) > 200 {
		return fmt.Errorf("known feature and actor of 1–200 bytes required: %w", ErrValidation)
	}
	if err := s.store.Set(ctx, string(key), enabled, actor); err != nil {
		return err
	}
	s.mu.Lock()
	s.expires = time.Time{}
	s.mu.Unlock()
	return nil
}
