package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"resume-builder/backend/internal/modules/jobs/internal/service"
)

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.Canceled):
		// The client is gone; do not manufacture a 500 or try to send a body.
		slog.Info("request_work_canceled")
	case errors.Is(err, context.DeadlineExceeded):
		slog.Warn("request_work_timed_out")
		http.Error(w, "request timed out", http.StatusGatewayTimeout)
	case errors.Is(err, service.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, service.ErrValidation):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		slog.Error("internal_error", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
