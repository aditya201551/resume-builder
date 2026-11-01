package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"resume-builder/backend/internal/jobmap"
)

func logJobError(ctx context.Context, event string, err error, attrs ...any) {
	level := slog.LevelError
	if errors.Is(err, context.Canceled) || ctx.Err() == context.Canceled {
		level = slog.LevelInfo
	}
	if errors.Is(err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded {
		level = slog.LevelWarn
	}
	var response *jobmap.FeedHTTPError
	if errors.As(err, &response) {
		attrs = append(attrs, "http_status", response.StatusCode, "retry_after_seconds", response.RetryAfter.Seconds())
		if response.StatusCode == http.StatusTooManyRequests {
			level = slog.LevelWarn
		}
	}
	attrs = append(attrs, "error", err)
	slog.Log(ctx, level, event, attrs...)
}

func (s *JobService) releaseSearch(ctx context.Context, key, source string) {
	cleanup, cancel := jobCleanupContext(ctx)
	defer cancel()
	if err := s.repo.ReleaseSearch(cleanup, key); err != nil {
		logJobError(cleanup, "job_lease_release_failed", err, "source", source)
	}
}

func jobCleanupContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
}
