package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPErrorLogsUseRoutePatternsAndPreserveStreaming(t *testing.T) {
	previous := slog.Default()
	var logs bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	mux := http.NewServeMux()
	mux.HandleFunc("GET /jobs/{id}", func(w http.ResponseWriter, r *http.Request) { http.Error(w, "failed", 503) })
	mux.HandleFunc("GET /stream", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("logging broke streaming")
		}
		w.Write([]byte("event: ready\n\n"))
		flusher.Flush()
	})
	handler := LogErrors(mux)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/jobs/private-id?token=private-token", nil))
	var event map[string]any
	if err := json.Unmarshal(logs.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	if event["level"] != "ERROR" || event["route"] != "GET /jobs/{id}" || event["status"] != float64(503) {
		t.Fatal(event)
	}
	if strings.Contains(logs.String(), "private-") {
		t.Fatal("logged request details")
	}
	logs.Reset()
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/stream", nil))
	if w.Code != 200 || !w.Flushed || !strings.Contains(w.Body.String(), "event: ready") || logs.Len() != 0 {
		t.Fatal("stream behavior changed", w, logs.String())
	}
}
