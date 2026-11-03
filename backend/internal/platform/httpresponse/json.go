package httpresponse

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
)

func JSON(w http.ResponseWriter, status int, v any) {
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
