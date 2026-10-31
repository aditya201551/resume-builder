package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type responseProbe struct {
	http.ResponseWriter
	written bool
}

func (w *responseProbe) WriteHeader(code int) { w.written = true; w.ResponseWriter.WriteHeader(code) }
func (w *responseProbe) Write(b []byte) (int, error) {
	w.written = true
	return w.ResponseWriter.Write(b)
}

func TestCanceledErrorsDoNotWriteServerErrors(t *testing.T) {
	w := &responseProbe{ResponseWriter: httptest.NewRecorder()}
	writeError(w, fmt.Errorf("database query interrupted: %w", context.Canceled))
	if w.written {
		t.Fatal("canceled client received a manufactured error response")
	}
}

func TestTimeoutAndServerFailuresRemainDistinct(t *testing.T) {
	for _, test := range []struct {
		err    error
		status int
	}{{context.DeadlineExceeded, 504}, {errors.New("database unavailable"), 500}} {
		w := httptest.NewRecorder()
		writeError(w, test.err)
		if w.Code != test.status {
			t.Fatalf("got %d, want %d", w.Code, test.status)
		}
	}
}
