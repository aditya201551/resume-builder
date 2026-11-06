package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"resume-builder/backend/internal/features"
)

type featureDecision bool

func (decision featureDecision) Enabled(context.Context, features.Key) bool {
	return bool(decision)
}

func TestRequireFeatureKeepsBackendEnforcementAndPrivateDecisions(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		called := false
		handler := RequireFeature(featureDecision(enabled), "test.enabled")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		}))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/test-feature", nil))
		if called != enabled || response.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("feature boundary did not enforce availability or private caching")
		}
		if enabled && response.Code != http.StatusNoContent {
			t.Fatal("enabled feature was blocked")
		}
		if !enabled && (response.Code != http.StatusNotFound || !strings.Contains(response.Body.String(), "feature_disabled")) {
			t.Fatal("disabled feature was exposed")
		}
	}
}
