package jobs

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"resume-builder/backend/internal/api/middleware"
	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/features"
)

type testFlags struct{ enabled atomic.Bool }

func (f *testFlags) Enabled(context.Context, features.Key) bool { return f.enabled.Load() }

func TestJobsRoutesAreAuthenticatedAndSwitchLive(t *testing.T) {
	flags := &testFlags{}
	var builds atomic.Int32
	module := &Module{flags: flags, build: func() (http.Handler, error) {
		builds.Add(1)
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), nil
	}}
	issuer := auth.NewJWTIssuer("test-only-secret", time.Hour)
	protect := middleware.RequireAuth(issuer)
	mux := http.NewServeMux()
	module.RegisterRoutes(func(pattern string, handler http.HandlerFunc) { mux.Handle(pattern, protect(handler)) })
	request := func(path string, authenticated bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		if authenticated {
			token, _ := issuer.Issue("owner")
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	if request("/api/jobs/countries", false).Code != 401 {
		t.Fatal("authentication was skipped")
	}
	for _, path := range []string{"/api/jobs", "/api/jobs/search", "/api/jobs/assist", "/api/jobs/countries", "/api/jobs/job_x/saved", "/api/jobs/unknown"} {
		w := request(path, true)
		if w.Code != 404 || !strings.Contains(w.Body.String(), "feature_disabled") || !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("disabled route %s: %d %s", path, w.Code, w.Body.String())
		}
	}
	if builds.Load() != 0 {
		t.Fatal("disabled Jobs initialized its dependencies")
	}
	flags.enabled.Store(true)
	var group sync.WaitGroup
	for range 20 {
		group.Go(func() {
			if request("/api/jobs/countries", true).Code != 204 {
				t.Error("could not enable Jobs live")
			}
		})
	}
	group.Wait()
	if builds.Load() != 1 {
		t.Fatal("concurrent requests initialized Jobs more than once")
	}
	flags.enabled.Store(false)
	if request("/api/jobs/countries", true).Code != 404 {
		t.Fatal("already initialized module bypassed its flag")
	}
}

func TestJobsInitializationFailureIsBoundedAndRetryable(t *testing.T) {
	flags := &testFlags{}
	flags.enabled.Store(true)
	builds := 0
	module := &Module{flags: flags, build: func() (http.Handler, error) {
		builds++
		return nil, errors.New("provider misconfigured")
	}}
	mux := http.NewServeMux()
	module.RegisterRoutes(func(pattern string, handler http.HandlerFunc) { mux.HandleFunc(pattern, handler) })
	for range 2 {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/jobs/countries", nil))
		if w.Code != 503 || strings.Contains(w.Body.String(), "misconfigured") {
			t.Fatal("initialization failure leaked or was not handled")
		}
	}
	if builds != 1 {
		t.Fatal("failed initialization was retried on every request")
	}
	module.retryAt = time.Time{}
	module.build = func() (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }), nil
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/jobs/countries", nil))
	if w.Code != 204 {
		t.Fatal("initialization did not recover")
	}
}
