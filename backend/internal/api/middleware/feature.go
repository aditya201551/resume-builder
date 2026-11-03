package middleware

import (
	"context"
	"net/http"

	"resume-builder/backend/internal/features"
	"resume-builder/backend/internal/platform/httpresponse"
)

type FeatureEvaluator interface {
	Enabled(context.Context, features.Key) bool
}

func RequireFeature(flags FeatureEvaluator, key features.Key) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "private, no-store")
			if !flags.Enabled(r.Context(), key) {
				httpresponse.JSON(w, http.StatusNotFound, map[string]string{"code": "feature_disabled", "message": "This feature is unavailable."})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
