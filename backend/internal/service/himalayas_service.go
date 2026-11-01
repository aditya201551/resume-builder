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

const himalayasCheckInterval = 10 * time.Minute

// A database lease spaces provider calls across workers, users and processes.
// Daily counters include filtered search pages and individual status checks.
func (s *JobService) reserveHimalayasRequest(ctx context.Context) error {
	retryAt, err := s.repo.ProviderRetryAt(ctx, "himalayas")
	if err != nil {
		return err
	}
	if retryAt != nil && retryAt.After(time.Now()) {
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
		if ctx.Err() != nil {
			s.releaseSearch(ctx, key, "himalayas-check")
		}
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
		err = s.repo.StoreSource(ctx, "himalayas", []jobmap.Job{current})
	}
	if err != nil {
		if ctx.Err() != nil {
			s.releaseSearch(ctx, key, "himalayas-check")
		}
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
