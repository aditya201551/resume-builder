package api_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"resume-builder/backend/internal/api"
	"resume-builder/backend/internal/api/handlers"
	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/features"
	"resume-builder/backend/internal/features/postgres"
)

func TestFeatureAPIAndRemovedRoutesWithSPA(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	mock.ExpectQuery("SELECT key, enabled FROM feature_flags").WillReturnRows(
		pgxmock.NewRows([]string{"key", "enabled"}).AddRow("retired.enabled", true),
	)
	flags := features.NewService(postgres.NewStore(mock))
	issuer := auth.NewJWTIssuer("test-only-secret", time.Hour)
	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<p>Resume app</p>"), 0600); err != nil {
		t.Fatal(err)
	}
	router := api.NewRouter(issuer, api.Handlers{
		Features: handlers.NewFeatureHandler(flags),
	}, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }, staticDir)
	request := func(method, path string, authenticated bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		if authenticated {
			token, _ := issuer.Issue("owner")
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	if request("GET", "/api/features", false).Code != 401 {
		t.Fatal("feature API bypassed auth")
	}
	w := request("GET", "/api/features", true)
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{}` || w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("unexpected feature snapshot: %d %s", w.Code, w.Body.String())
	}
	for _, route := range []struct{ method, path string }{
		{"POST", "/api/jobs/search"}, {"POST", "/api/jobs/assist"}, {"GET", "/api/jobs/countries"},
		{"GET", "/api/jobs/job_x"}, {"PUT", "/api/jobs/job_x/saved"}, {"DELETE", "/api/jobs/job_x/saved"},
	} {
		w := request(route.method, route.path, true)
		if w.Code != 404 || strings.Contains(w.Body.String(), "Resume app") {
			t.Fatalf("removed API route fell through to the app: %s", route.path)
		}
	}
	if request("GET", "/api/health", false).Code != 204 {
		t.Fatal("feature registry affected core application")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
