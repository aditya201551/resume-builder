package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"resume-builder/backend/internal/ai"
	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/features"
	"resume-builder/backend/internal/repository"
	"resume-builder/backend/internal/service"
	"strings"
	"testing"
)

type accessStore struct{ allowed bool }

func (s *accessStore) List(context.Context, string) ([]repository.AIConnection, error) {
	return []repository.AIConnection{}, nil
}
func (s *accessStore) Get(context.Context, string, string) (*repository.AIConnection, error) {
	return nil, repository.ErrNotFound
}
func (s *accessStore) Save(context.Context, repository.AIConnection) error   { return nil }
func (s *accessStore) Delete(context.Context, string, string) error          { return nil }
func (s *accessStore) Validate(context.Context, string, string, int64) error { return nil }
func (s *accessStore) Preference(context.Context, string) (*repository.AIPreference, error) {
	return nil, nil
}
func (s *accessStore) Select(context.Context, string, string, string) error { return nil }
func (s *accessStore) Allowed(context.Context, string) (bool, error)        { return s.allowed, nil }

type accessFlags map[string]bool

func (f accessFlags) List(context.Context) (map[string]bool, error) { return f, nil }
func testAIAccess(platform, byok, allowed bool) *AIHandler {
	svc := service.NewAISettingsService(&accessStore{allowed}, nil, &ai.Config{Provider: ai.ProviderAnthropic, APIKey: "synthetic-platform-key", Model: "claude-sonnet-4"}, "")
	return NewAIHandler(svc, features.NewService(accessFlags{"platform_ai": platform, "byok_ai": byok}), "http://localhost:5173")
}
func TestAIBoundariesRejectBeforeReadingBodyOrOpeningStream(t *testing.T) {
	for _, flags := range []struct{ platform, byok, allowed bool }{{false, false, true}, {true, false, false}, {false, true, false}} {
		h := testAIAccess(flags.platform, flags.byok, flags.allowed)
		for _, kind := range []string{"chat", "import", "test", "models", "save", "select"} {
			r := httptest.NewRequest(http.MethodPost, "/api/test", strings.NewReader("invalid request body"))
			r = r.WithContext(auth.ContextWithUserID(r.Context(), "owner"))
			r.SetPathValue("provider", "anthropic")
			w := httptest.NewRecorder()
			switch kind {
			case "chat":
				NewAgentHandler(h, 1024, nil, nil).Chat(w, r)
			case "import":
				NewResumeImportHandler(nil, h).Import(w, r)
			case "test":
				h.Test(w, r)
			case "models":
				h.Models(w, r)
			case "save":
				h.Save(w, r)
			case "select":
				h.Select(w, r)
			}
			if w.Code < 400 || w.Header().Get("Content-Type") == "text/event-stream" {
				t.Fatalf("%s bypassed gate: %d", kind, w.Code)
			}
		}
	}
}
func TestAIOriginProtection(t *testing.T) {
	h := testAIAccess(false, false, false)
	for _, tc := range []struct {
		origin string
		cookie bool
		status int
	}{{"https://evil.example", true, 403}, {"", true, 403}, {"http://localhost:5173", true, 204}, {"", false, 204}} {
		r := httptest.NewRequest("PUT", "/api/ai/preferences", nil)
		r.Header.Set("Origin", tc.origin)
		if tc.cookie {
			r.AddCookie(&http.Cookie{Name: "session", Value: "synthetic"})
		}
		w := httptest.NewRecorder()
		h.Protect(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })(w, r)
		if w.Code != tc.status {
			t.Fatalf("origin %q cookie %v: %d", tc.origin, tc.cookie, w.Code)
		}
	}
}

func TestPublicPlatformGateUsesBackendFlagSnapshot(t *testing.T) {
	for _, tc := range []struct{ platform, public, ready bool }{{true, true, true}, {true, false, false}, {false, true, false}} {
		svc := service.NewAISettingsService(&accessStore{allowed: false}, nil, &ai.Config{Provider: ai.ProviderAnthropic, APIKey: "synthetic", Model: "claude-sonnet-4"}, "")
		h := NewAIHandler(svc, features.NewService(accessFlags{"platform_ai": tc.platform, "platform_ai_public": tc.public}), "http://localhost:5173")
		for _, pdf := range []bool{false, true} {
			r := httptest.NewRequest("POST", "/api/test", nil)
			r = r.WithContext(auth.ContextWithUserID(r.Context(), "non-invited-user"))
			w := httptest.NewRecorder()
			cfg, ready := h.Resolve(w, r, pdf)
			if ready != tc.ready || ready && cfg.APIKey != "synthetic" {
				t.Fatalf("wrong backend policy for platform=%v public=%v pdf=%v", tc.platform, tc.public, pdf)
			}
		}
	}
}
