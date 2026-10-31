package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"resume-builder/backend/internal/service"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		if request, ok := w.(interface{ RequestContext() context.Context }); ok && request.RequestContext().Err() != nil {
			slog.Info("http_response_canceled")
			return
		}
		slog.Error("http_response_encode_failed", "status", status, "error", err)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return false
	}
	return true
}

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
