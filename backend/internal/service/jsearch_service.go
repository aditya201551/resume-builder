package service

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"strings"
	"time"

	"resume-builder/backend/internal/jobmap"
)

// A user search only waits for the database claim, never the provider. The
// bounded background request survives browser cancellation and stops on shutdown.
func (s *JobService) queueExternal(ctx context.Context, search jobmap.Search) bool {
	query, country := jobmap.ExternalSearch(search)
	if query == "" || ctx.Err() != nil {
		return false
	}
	select {
	case s.externalGate <- struct{}{}:
	default:
		return false
	}
	key := "jsearch:" + strings.ToLower(country+":"+query)
	token := rand.Text()
	claimed, err := s.repo.ClaimExternalSearch(ctx, key, token, time.Minute, 24*time.Hour)
	if err != nil || !claimed {
		<-s.externalGate
		if err != nil {
			logJobError(ctx, "jsearch_lease_failed", err)
		} else {
			slog.Debug("jsearch_skipped", "reason", "cached_or_running")
		}
		return false
	}
	s.externalRefreshing.Store(true)
	started := s.runBackground(func(ctx context.Context) {
		defer func() { s.externalRefreshing.Store(false); <-s.externalGate }()
		work, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		s.searchExternal(work, query, country, key, token)
	})
	if !started {
		s.externalRefreshing.Store(false)
		<-s.externalGate
		s.releaseExternal(ctx, key, token)
	}
	return started
}

func (s *JobService) releaseExternal(ctx context.Context, key, token string) {
	cleanup, cancel := jobCleanupContext(ctx)
	defer cancel()
	if err := s.repo.ReleaseExternalSearch(cleanup, key, token); err != nil {
		logJobError(cleanup, "jsearch_lease_release_failed", err)
	}
}

func (s *JobService) searchExternal(ctx context.Context, query, country, key, token string) {
	// Cleanup is token-scoped, including shutdown and timeout paths. A failed
	// process can only block this query for one minute, never the 24-hour cache TTL.
	defer s.releaseExternal(ctx, key, token)
	reserved, err := s.repo.ReserveRequest(ctx, "jsearch", time.Now().UTC().Format("2006-01"), s.cfg.JSearchLimit)
	if err != nil || !reserved {
		if err != nil {
			logJobError(ctx, "jsearch_budget_failed", err)
			s.storeSourceStatus(ctx, jobmap.SourceStatus{ID: "jsearch", Name: "JSearch", Kind: "aggregator", Status: "error", Message: "Broad search is unavailable. Showing available indexed listings."})
		} else {
			slog.Warn("jsearch_quota_exhausted", "monthly_limit", s.cfg.JSearchLimit)
			s.storeSourceStatus(ctx, jobmap.SourceStatus{ID: "jsearch", Name: "JSearch", Kind: "aggregator", Status: "quota_exhausted", Message: "Broad search has reached its prototype request budget. Cached and company-feed jobs remain available."})
		}
		s.backoffExternal(ctx, key, token, err)
		return
	}
	started := time.Now()
	slog.Info("jsearch_started", "country", country)
	s.storeSourceStatus(ctx, jobmap.SourceStatus{ID: "jsearch", Name: "JSearch", Kind: "aggregator", Status: "refreshing"})
	jobs, err := s.client.FetchJSearch(ctx, query, country)
	valid := []jobmap.Job{}
	now := time.Now().UTC()
	if err == nil {
		for _, j := range jobs {
			if job, ok := jobmap.Prepare(j, now); ok {
				valid = append(valid, job)
			}
		}
		err = s.repo.StoreSource(ctx, "jsearch", valid, false)
	}
	if err == nil {
		err = s.repo.CompleteExternalSearch(ctx, key, token)
	}
	if err == nil {
		s.storeSourceStatus(ctx, jobmap.SourceStatus{ID: "jsearch", Name: "JSearch", Kind: "aggregator", Status: "ready", JobCount: len(valid), RefreshedAt: &now})
		slog.Info("jsearch_completed", "received", len(jobs), "accepted", len(valid), "duration_ms", time.Since(started).Milliseconds())
		return
	}
	logJobError(ctx, "jsearch_failed", err, "duration_ms", time.Since(started).Milliseconds())
	s.backoffExternal(ctx, key, token, err)
	s.storeSourceStatus(ctx, jobmap.SourceStatus{ID: "jsearch", Name: "JSearch", Kind: "aggregator", Status: "error", Message: "Broad search was interrupted or unavailable. Cached listings remain available; the next search can retry."})
}

func (s *JobService) backoffExternal(ctx context.Context, key, token string, cause error) {
	if s.workCtx.Err() != nil {
		return
	} // shutdown permits an immediate retry on the next instance
	delay := 5 * time.Minute
	var response *jobmap.FeedHTTPError
	if errors.As(cause, &response) {
		delay = max(delay, response.RetryAfter)
	}
	until := time.Now().UTC().Add(delay)
	cleanup, cancel := jobCleanupContext(ctx)
	defer cancel()
	if err := s.repo.FailExternalSearch(cleanup, key, token, until); err != nil {
		logJobError(cleanup, "jsearch_backoff_failed", err)
		return
	}
	slog.Warn("jsearch_retry_deferred", "retry_at", until)
}
