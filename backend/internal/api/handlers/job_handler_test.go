package handlers

import (
	"net/http/httptest"
	"resume-builder/backend/internal/jobsearch"
	"strings"
	"testing"
)

func TestDiscoveryRejectsUnsupportedHardFilters(t *testing.T) {
	for _, body := range []string{`{"intent":{"version":1,"salary":100000}}`, `{"intent":{"version":1},"invented":true}`, `{} {}`} {
		r := httptest.NewRequest("POST", "/api/jobs/search", strings.NewReader(body))
		w := httptest.NewRecorder()
		var request jobsearch.Request
		if decodeJobRequest(w, r, &request) || w.Code != 400 {
			t.Fatal("unsupported request accepted", body, w.Code)
		}
	}
}
