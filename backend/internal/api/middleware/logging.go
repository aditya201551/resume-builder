package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// LogErrors records failed HTTP responses, including direct http.Error calls.
// Route patterns are logged rather than URLs, queries, bodies or cookies.
func LogErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		tracked := &loggedResponse{ResponseWriter: w, ctx: r.Context()}
		next.ServeHTTP(tracked, r)
		if r.Context().Err() == context.Canceled {
			slog.InfoContext(r.Context(), "http_request_canceled", "method", r.Method, "route", r.Pattern, "response_status", tracked.status, "duration_ms", time.Since(started).Milliseconds())
			return
		}
		if tracked.status < 400 {
			return
		}
		level := slog.LevelWarn
		if tracked.status >= 500 {
			level = slog.LevelError
		}
		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		slog.Log(r.Context(), level, "http_request_failed", "method", r.Method, "route", route, "status", tracked.status, "duration_ms", time.Since(started).Milliseconds())
	})
}

type loggedResponse struct {
	http.ResponseWriter
	status int
	ctx    context.Context
}

func (w *loggedResponse) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	if status < 200 {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *loggedResponse) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(body)
	if err != nil {
		w.logWriteError("http_write_failed", err)
	}
	return n, err
}
func (w *loggedResponse) Unwrap() http.ResponseWriter     { return w.ResponseWriter }
func (w *loggedResponse) RequestContext() context.Context { return w.ctx }

// Preserve flushing for the AI assistant's streaming responses.
func (w *loggedResponse) Flush() {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if err := http.NewResponseController(w.ResponseWriter).Flush(); err != nil {
		w.logWriteError("http_flush_failed", err)
	}
}

func (w *loggedResponse) logWriteError(event string, err error) {
	if w.ctx != nil && w.ctx.Err() != nil {
		slog.Info(event, "reason", "request_canceled")
		return
	}
	slog.Warn(event, "error", err)
}
