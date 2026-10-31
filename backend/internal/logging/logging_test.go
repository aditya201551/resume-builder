package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestJSONLoggingRedactsErrorCredentials(t *testing.T) {
	t.Setenv("TEST_API_KEY", "secret-value-123")
	var out bytes.Buffer
	logger := slog.New(newHandler(&out, slog.LevelInfo))
	logger.Info("api_listening", "port", "8080")
	logger.Error("provider_failed", "error", fmt.Errorf("credential %s", "secret-value-123"))
	var first, second map[string]any
	decoder := json.NewDecoder(&out)
	if err := decoder.Decode(&first); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Decode(&second); err != nil {
		t.Fatal(err)
	}
	if first["level"] != "INFO" || second["level"] != "ERROR" || second["error"] != "credential [redacted]" {
		t.Fatal(first, second)
	}
}

func TestProviderErrorsExcludeURLQueriesAndOAuthBodies(t *testing.T) {
	request := &url.Error{Op: "Get", URL: "https://example.com/?apiKey=private-key", Err: errors.New("connection refused")}
	if got := safeError(fmt.Errorf("upstream failed: %w", request)); got != "connection refused" {
		t.Fatal(got)
	}
	retrieve := &oauth2.RetrieveError{Response: &http.Response{StatusCode: 401}, Body: []byte(`client_secret=private-key`)}
	if got := safeError(retrieve); got != "OAuth provider returned HTTP 401" || strings.Contains(got, "private-key") {
		t.Fatal(got)
	}
}
