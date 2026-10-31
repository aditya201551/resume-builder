package service

import (
	"context"
	"log/slog"
	"time"
)

// Start refreshes employer feeds only. The full Himalayas crawl is paused while
// the jobs module is redesigned; cached listings and individual checks remain.
// The ticker starts before the first pass, so a long pass does not add an idle delay.
func (s *JobService) Start(ctx context.Context) {
	s.workMu.Lock()
	if s.started || s.stopping {
		s.workMu.Unlock()
		return
	}
	s.started = true
	s.workMu.Unlock()
	slog.Info("job_workers_started", "employer_sources", len(s.cfg.Companies), "employer_refresh_interval", s.cfg.RefreshInterval.String(), "himalayas_feed_enabled", false, "schedule_check_interval", "1m", "himalayas_key_configured", s.cfg.HimalayasKey != "", "jsearch_key_configured", s.cfg.JSearchKey != "")
	context.AfterFunc(ctx, func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := s.Stop(cleanup); err != nil {
			logJobError(cleanup, "job_workers_stop_failed", err)
		}
	})
	s.runBackground(func(ctx context.Context) {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		if ctx.Err() != nil {
			return
		}
		s.refresh(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.refresh(ctx)
			}
		}
	})
}

func (s *JobService) runBackground(work func(context.Context)) bool {
	s.workMu.Lock()
	defer s.workMu.Unlock()
	if s.stopping {
		return false
	}
	s.workWG.Add(1)
	go func() { defer s.workWG.Done(); work(s.workCtx) }()
	return true
}

// Stop cancels background requests and waits for bounded, uncanceled DB cleanup.
// Launches and Wait are serialized so shutdown cannot race with a new search.
func (s *JobService) Stop(ctx context.Context) error {
	s.workMu.Lock()
	if !s.stopping {
		s.stopping = true
		s.cancelWork()
		go func() { s.workWG.Wait(); close(s.stopped) }()
	}
	s.workMu.Unlock()
	select {
	case <-s.stopped:
		return nil
	default:
	}
	select {
	case <-s.stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
