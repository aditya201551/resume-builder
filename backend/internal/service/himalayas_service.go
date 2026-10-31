package service

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	if err != nil || !claimed {
		return
	}
	defer s.repo.ReleaseSearch(ctx, "himalayas-feed-refresh")
	p, err := s.repo.FeedProgress(ctx, "himalayas")
	if err != nil {
		log.Printf("Himalayas checkpoint: %v", err)
		return
	}
	if p.RetryAt != nil && p.RetryAt.After(time.Now()) {
		return
	}
	if p.Cursor == "" && p.CompletedAt != nil && time.Since(*p.CompletedAt) < himalayasRefreshInterval {
		return
	}
	if p.Cursor == "" {
		p.JobCount = 0
	}
	status := jobmap.SourceStatus{ID: "himalayas", Name: "Himalayas", Kind: "aggregator", Status: "refreshing", JobCount: p.JobCount, RefreshedAt: p.CompletedAt}
	if err := s.repo.StoreStatus(ctx, status); err != nil {
		return
	}
	workCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	seen := map[string]bool{p.Cursor: true}
	for {
		err = s.reserveHimalayasRequest(workCtx)
		if err != nil {
			break
		}
		var jobs []jobmap.Job
		var cursor string
		jobs, cursor, err = s.client.FetchHimalayasPage(workCtx, p.Cursor)
		if err != nil {
			s.himalayasBackoff(ctx, err)
			var response *jobmap.FeedHTTPError
			if errors.As(err, &response) && response.StatusCode == http.StatusBadRequest {
				// An expired cursor restarts the next walk without deleting any jobs.
				p.Cursor, p.JobCount, p.CompletedAt = "", 0, nil
				if saveErr := s.repo.StoreFeedPage(ctx, "himalayas", nil, p); saveErr != nil {
					log.Printf("Himalayas reset: %v", saveErr)
				}
			}
			break
		}
		if cursor != "" && seen[cursor] {
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
		if err = s.repo.StoreFeedPage(workCtx, "himalayas", valid, p); err != nil {
			break
		}
		status.JobCount = p.JobCount
		if cursor == "" {
			status.Status, status.RefreshedAt = "ready", p.CompletedAt
			break
		}
		seen[cursor] = true
	}
	if err != nil {
		status.Status = "error"
		status.Message = "Remote feed sync paused. Collected listings remain available; the next attempt resumes from its saved page."
		log.Printf("Himalayas sync: %v", err)
	}
	if err := s.repo.StoreStatus(ctx, status); err != nil {
		log.Printf("Himalayas status: %v", err)
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
		if err := s.repo.DelayFeed(ctx, "himalayas", time.Now().UTC().Add(delay)); err != nil {
			log.Printf("Himalayas backoff: %v", err)
		}
	}
}

func (s *JobService) verifyHimalayas(ctx context.Context, userID string, cached jobmap.Job) (jobmap.Job, error) {
	if cached.CheckedAt != nil && time.Since(*cached.CheckedAt) < himalayasCheckInterval {
		return cached, nil
	}
	key := "himalayas-check:" + cached.ID
	claimed, err := s.repo.ClaimSearch(ctx, key, himalayasCheckInterval)
	if err != nil {
		return cached, err
	}
	if !claimed {
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
			cached.CheckFailed = true
			return cached, nil
		}
		current.CheckedAt = &now
		err = s.repo.StoreSource(ctx, "himalayas", []jobmap.Job{current}, false)
	}
	if err != nil {
		return cached, err
	}
	return s.repo.Get(ctx, userID, cached.ID)
}
