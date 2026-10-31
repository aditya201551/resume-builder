// Package logging configures structured server logs without request payloads or credentials.
package logging

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"golang.org/x/oauth2"
)

// Configure writes JSON levels to stdout so platforms do not classify every
// standard-library stderr line as an error. LOG_LEVEL optionally enables debug.
func Configure() error {
	var level slog.LevelVar
	slog.SetDefault(slog.New(newHandler(os.Stdout, &level)))
	if raw := os.Getenv("LOG_LEVEL"); raw != "" {
		switch strings.ToLower(raw) {
		case "debug":
			level.Set(slog.LevelDebug)
		case "info":
			level.Set(slog.LevelInfo)
		case "warn":
			level.Set(slog.LevelWarn)
		case "error":
			level.Set(slog.LevelError)
		default:
			return fmt.Errorf("LOG_LEVEL must be debug, info, warn or error")
		}
	}
	return nil
}

func newHandler(out io.Writer, level slog.Leveler) *slog.JSONHandler {
	return slog.NewJSONHandler(out, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if err, ok := a.Value.Any().(error); ok {
				a.Value = slog.StringValue(safeError(err))
			}
			return a
		},
	})
}

func safeError(err error) string {
	var retrieve *oauth2.RetrieveError
	if errors.As(err, &retrieve) {
		if retrieve.Response != nil {
			return fmt.Sprintf("OAuth provider returned HTTP %d", retrieve.Response.StatusCode)
		}
		return "OAuth provider request failed"
	}
	var request *url.Error
	if errors.As(err, &request) {
		return safeError(request.Err)
	}
	message := err.Error()
	for _, entry := range os.Environ() {
		name, value, ok := strings.Cut(entry, "=")
		if !ok || len(value) < 4 {
			continue
		}
		name = strings.ToUpper(name)
		if strings.Contains(name, "SECRET") || strings.Contains(name, "PASSWORD") || strings.Contains(name, "API_KEY") || strings.Contains(name, "TOKEN") || name == "DATABASE_URL" {
			message = strings.ReplaceAll(message, value, "[redacted]")
			message = strings.ReplaceAll(message, url.QueryEscape(value), "[redacted]")
		}
	}
	if len(message) > 2048 {
		message = message[:2048] + " [truncated]"
	}
	return message
}
