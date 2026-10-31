package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"resume-builder/backend/internal/jobmap"
)

const himalayasRefreshInterval = 24 * time.Hour
const himalayasCheckInterval = 10 * time.Minute

// A pass is shorter than its shared lease. Large feeds resume on the next pass
// instead of discarding progress or holding a transaction during network calls.
func (s *JobService) refreshHimalayas(ctx context.Context) {
	claimed, err := s.repo.ClaimSearch(ctx, "himalayas-feed-refresh", 15*time.Minute)
	if err != nil {
		logJobError(ctx, "himalayas_sync_lease_failed", err)
		return
	}
	if !claimed {
		slog.Info("himalayas_sync_skipped", "reason", "refresh_lease_active")
		return
	}
	defer s.releaseSearch(ctx, "himalayas-feed-refresh", "himalayas")
	p, err := s.repo.FeedProgress(ctx, "himalayas")
	if err != nil {
		logJobError(ctx, "himalayas_checkpoint_failed", err)
		return
	}
	if p.RetryAt != nil && p.RetryAt.After(time.Now()) {
		slog.Info("himalayas_sync_skipped", "reason", "provider_backoff", "retry_at", *p.RetryAt)
		return
	}
	if p.Cursor == "" && p.CompletedAt != nil && time.Since(*p.CompletedAt) < himalayasRefreshInterval {
		slog.Info("himalayas_sync_skipped", "reason", "not_due", "completed_at", *p.CompletedAt, "next_sync_at", p.CompletedAt.Add(himalayasRefreshInterval))
		return
	}
	if p.Cursor == "" {
		p.JobCount = 0
	}
	status := jobmap.SourceStatus{ID: "himalayas", Name: "Himalayas", Kind: "aggregator", Status: "refreshing", JobCount: p.JobCount, RefreshedAt: p.CompletedAt}
	if err := s.repo.StoreStatus(ctx, status); err != nil {
		logJobError(ctx, "himalayas_status_failed", err, "status", "refreshing")
		return
	}
	started := time.Now()
	pages, received, accepted := 0, 0, 0
	slog.Info("himalayas_sync_started", "resuming", p.Cursor != "", "previously_collected", p.JobCount, "pass_timeout", "10m")
	workCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	seen := map[string]bool{p.Cursor: true}
	operation := "reserve_request"
	for {
		operation = "reserve_request"
		err = s.reserveHimalayasRequest(workCtx)
		if err != nil {
			break
		}
		var jobs []jobmap.Job
		var cursor string
		operation = "fetch_page"
		jobs, cursor, err = s.client.FetchHimalayasPage(workCtx, p.Cursor)
		if err != nil {
			s.himalayasBackoff(ctx, err)
			var response *jobmap.FeedHTTPError
			if errors.As(err, &response) && response.StatusCode == http.StatusBadRequest {
				// An expired cursor restarts the next walk without deleting any jobs.
				p.Cursor, p.JobCount, p.CompletedAt = "", 0, nil
				if saveErr := s.repo.StoreFeedPage(ctx, "himalayas", nil, p); saveErr != nil {
					logJobError(ctx, "himalayas_cursor_reset_failed", saveErr)
				} else {
					slog.Warn("himalayas_cursor_reset", "reason", "invalid_or_expired_cursor")
				}
			}
			break
		}
		if cursor != "" && seen[cursor] {
			operation = "validate_cursor"
			err = fmt.Errorf("Himalayas cursor cycle")
			break
		}
		now := time.Now().UTC()
		valid := []jobmap.Job{}
		for _, job := range jobs {
			if j, ok := jobmap.Prepare(job, now); ok {
				valid = append(valid, j)
			}
		}
		p.Cursor = cursor
		p.JobCount += len(valid)
		if cursor == "" {
			p.CompletedAt = &now
		}
		operation = "store_page"
		if err = s.repo.StoreFeedPage(workCtx, "himalayas", valid, p); err != nil {
			break
		}
		status.JobCount = p.JobCount
		pages++
		received += len(jobs)
		accepted += len(valid)
		if pages == 1 || pages%25 == 0 {
			slog.Info("himalayas_sync_progress", "pages", pages, "received", received, "accepted", accepted, "collected_total", p.JobCount, "has_more", cursor != "", "duration_ms", time.Since(started).Milliseconds())
		}
		if cursor == "" {
			status.Status, status.RefreshedAt = "ready", p.CompletedAt
			slog.Info("himalayas_sync_completed", "pages", pages, "received", received, "accepted", accepted, "collected_total", p.JobCount, "duration_ms", time.Since(started).Milliseconds(), "next_sync_at", p.CompletedAt.Add(himalayasRefreshInterval))
			break
		}
		seen[cursor] = true
	}
	if err != nil {
		status.Status = "error"
		status.Message = "Remote feed sync paused. Collected listings remain available; the next attempt resumes from its saved page."
		logJobError(workCtx, "himalayas_sync_paused", err, "operation", operation, "pages_committed", pages, "collected_total", status.JobCount, "duration_ms", time.Since(started).Milliseconds())
	}
	if err := s.repo.StoreStatus(ctx, status); err != nil {
		logJobError(ctx, "himalayas_status_failed", err, "status", status.Status)
	}
}

// A database lease spaces provider calls across workers, users and processes.
// Daily counters include both feed pages and individual status checks.
func (s *JobService) reserveHimalayasRequest(ctx context.Context) error {
	p, err := s.repo.FeedProgress(ctx, "himalayas")
	if err != nil {
		return err
	}
	if p.RetryAt != nil && p.RetryAt.After(time.Now()) {
		return fmt.Errorf("Himalayas provider backoff is active")
	}
	for {
		allowed, err := s.repo.ClaimSearch(ctx, "himalayas-request-slot", 1100*time.Millisecond)
		if err != nil {
			return err
		}
		if allowed {
			break
		}
		timer := time.NewTimer(1100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	now := time.Now().UTC()
	allowed, err := s.repo.ReserveRequest(ctx, "himalayas", now.Format("2006-01-02"), 10000)
	if err != nil {
		return err
	}
	if !allowed {
		until := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
		if err := s.repo.DelayFeed(ctx, "himalayas", until); err != nil {
			return err
		}
		slog.Warn("himalayas_quota_exhausted", "daily_limit", 10000, "retry_at", until)
		return fmt.Errorf("Himalayas daily request budget exhausted")
	}
	return nil
}

func (s *JobService) himalayasBackoff(ctx context.Context, err error) {
	var response *jobmap.FeedHTTPError
	if errors.As(err, &response) && response.StatusCode == http.StatusTooManyRequests {
		delay := response.RetryAfter
		if delay <= 0 {
			delay = 15 * time.Minute
		}
		until := time.Now().UTC().Add(delay)
		if err := s.repo.DelayFeed(ctx, "himalayas", until); err != nil {
			logJobError(ctx, "himalayas_backoff_save_failed", err)
		} else {
			slog.Warn("himalayas_rate_limited", "retry_at", until, "retry_after_seconds", delay.Seconds())
		}
	}
}

func (s *JobService) verifyHimalayas(ctx context.Context, userID string, cached jobmap.Job) (jobmap.Job, error) {
	if cached.CheckedAt != nil && time.Since(*cached.CheckedAt) < himalayasCheckInterval {
		slog.Debug("himalayas_check_skipped", "reason", "recently_checked", "job_id", cached.ID)
		return cached, nil
	}
	key := "himalayas-check:" + cached.ID
	claimed, err := s.repo.ClaimSearch(ctx, key, himalayasCheckInterval)
	if err != nil {
		logJobError(ctx, "himalayas_check_lease_failed", err, "job_id", cached.ID)
		return cached, err
	}
	if !claimed {
		slog.Debug("himalayas_check_skipped", "reason", "check_lease_active", "job_id", cached.ID)
		// An in-flight or failed check cannot claim that cached availability is fresh.
		cached.CheckFailed = true
		return cached, nil
	}
	checkCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	err = s.reserveHimalayasRequest(checkCtx)
	var current jobmap.Job
	var closed bool
	if err == nil {
		current, closed, err = s.client.FetchHimalayasJob(checkCtx, cached.ProviderID)
	}
	if err != nil {
		s.himalayasBackoff(ctx, err)
		logJobError(checkCtx, "himalayas_check_failed", err, "job_id", cached.ID, "using_cached", true)
		cached.CheckFailed = true
		return cached, nil // outages never close a job or erase its bookmark
	}
	now := time.Now().UTC()
	if closed {
		err = s.repo.MarkJobClosed(ctx, cached.ID, now)
	} else {
		var ok bool
		current, ok = jobmap.Prepare(current, now)
		if !ok {
			slog.Warn("himalayas_check_invalid_job", "job_id", cached.ID, "using_cached", true)
			cached.CheckFailed = true
			return cached, nil
		}
		current.CheckedAt = &now
		err = s.repo.StoreSource(ctx, "himalayas", []jobmap.Job{current}, false)
	}
	if err != nil {
		logJobError(ctx, "himalayas_check_save_failed", err, "job_id", cached.ID, "closed", closed)
		return cached, err
	}
	current, err = s.repo.Get(ctx, userID, cached.ID)
	if err != nil {
		logJobError(ctx, "himalayas_check_read_failed", err, "job_id", cached.ID)
		return current, err
	}
	slog.Info("himalayas_check_completed", "job_id", cached.ID, "closed", closed)
	return current, nil
}
