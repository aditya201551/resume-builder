package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"resume-builder/backend/internal/ai"
	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/features"
	"resume-builder/backend/internal/platform/httpresponse"
	"resume-builder/backend/internal/service"
)

// AIHandler is the shared module boundary: flags are evaluated here, never in
// provider clients. Counters bound platform resources even for personal keys.
type AIHandler struct {
	svc     *service.AISettingsService
	flags   *features.Service
	origin  string
	mu      sync.Mutex
	active  map[string]int
	total   int
	windows map[string]aiRateWindow
}
type aiRateWindow struct {
	until time.Time
	count int
}

func NewAIHandler(svc *service.AISettingsService, flags *features.Service, origin string) *AIHandler {
	return &AIHandler{svc: svc, flags: flags, origin: strings.TrimRight(origin, "/"), active: make(map[string]int), windows: make(map[string]aiRateWindow)}
}
func (h *AIHandler) availability(ctx context.Context) service.AIAvailability {
	flags := h.flags.Snapshot(ctx)
	return service.AIAvailability{Platform: flags[features.PlatformAI], BYOK: flags[features.BYOKAI], PlatformPublic: flags[features.PlatformAIPublic]}
}
func (h *AIHandler) Resolve(w http.ResponseWriter, r *http.Request, pdf bool) (*ai.Config, bool) {
	user, _ := auth.UserIDFromContext(r.Context())
	flags := h.availability(r.Context())
	if !flags.Platform && !flags.BYOK {
		writeAIError(w, &service.AIError{Code: "ai_disabled", Message: "AI is currently disabled.", Status: 403})
		return nil, false
	}
	cfg, err := h.svc.Resolve(r.Context(), user, flags, pdf)
	if err != nil {
		writeAIError(w, err)
		return nil, false
	}
	return cfg, true
}
func (h *AIHandler) Begin(w http.ResponseWriter, r *http.Request) (func(), bool) {
	user, _ := auth.UserIDFromContext(r.Context())
	h.mu.Lock()
	now := time.Now()
	if len(h.windows) >= 4096 {
		for id, window := range h.windows {
			if now.After(window.until) {
				delete(h.windows, id)
			}
		}
	}
	window, exists := h.windows[user]
	if !exists && len(h.windows) >= 4096 || now.Before(window.until) && window.count >= 20 {
		h.mu.Unlock()
		w.Header().Set("Retry-After", "60")
		writeAIError(w, &service.AIError{Code: "ai_rate_limited", Message: "Too many AI requests. Try again in a minute.", Status: 429})
		return nil, false
	}
	if h.active[user] >= 2 || h.total >= 8 {
		h.mu.Unlock()
		writeAIError(w, &service.AIError{Code: "ai_busy", Message: "AI is busy. Try again shortly.", Status: 429})
		return nil, false
	}
	h.active[user]++
	if !now.Before(window.until) {
		window = aiRateWindow{until: now.Add(time.Minute)}
	}
	window.count++
	h.windows[user] = window
	h.total++
	h.mu.Unlock()
	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.total--
		h.active[user]--
		if h.active[user] == 0 {
			delete(h.active, user)
		}
	}, true
}

// Cookie-authenticated mutations require the configured frontend origin. Bearer
// clients without a session cookie remain usable without browser Origin headers.
func (h *AIHandler) Protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		if r.Method != http.MethodGet {
			if cookie, err := r.Cookie("session"); err == nil && cookie.Value != "" {
				if r.Header.Get("Origin") != h.origin {
					writeAIError(w, &service.AIError{Code: "origin_rejected", Message: "Request origin is not allowed.", Status: 403})
					return
				}
			}
		}
		next(w, r)
	}
}
func writeAIError(w http.ResponseWriter, err error) {
	var failure *service.AIError
	if !errors.As(err, &failure) {
		failure = &service.AIError{Code: "ai_unavailable", Message: "AI settings are temporarily unavailable. Try again.", Status: 503}
	}
	httpresponse.JSON(w, failure.Status, failure)
}
func (h *AIHandler) Settings(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserIDFromContext(r.Context())
	result, err := h.svc.Settings(r.Context(), user, h.availability(r.Context()))
	if err != nil {
		writeAIError(w, err)
		return
	}
	httpresponse.JSON(w, 200, result)
}
func (h *AIHandler) Save(w http.ResponseWriter, r *http.Request) {
	flags := h.availability(r.Context())
	if err := h.svc.RequireBYOK(flags); err != nil {
		writeAIError(w, err)
		return
	}
	release, ok := h.Begin(w, r)
	if !ok {
		return
	}
	defer release()
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	var body struct {
		APIKey string `json:"api_key"`
		Model  string `json:"model"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	user, _ := auth.UserIDFromContext(ctx)
	if err := h.svc.Save(ctx, user, r.PathValue("provider"), body.APIKey, body.Model, flags); err != nil {
		writeAIError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (h *AIHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserIDFromContext(r.Context())
	if err := h.svc.Delete(r.Context(), user, r.PathValue("provider")); err != nil {
		writeAIError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (h *AIHandler) Models(w http.ResponseWriter, r *http.Request) {
	release, ok := h.Begin(w, r)
	if !ok {
		return
	}
	defer release()
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	user, _ := auth.UserIDFromContext(ctx)
	models, err := h.svc.Models(ctx, user, r.PathValue("provider"), h.availability(ctx))
	if err != nil {
		writeAIError(w, err)
		return
	}
	httpresponse.JSON(w, 200, models)
}
func (h *AIHandler) Test(w http.ResponseWriter, r *http.Request) {
	release, ok := h.Begin(w, r)
	if !ok {
		return
	}
	defer release()
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	user, _ := auth.UserIDFromContext(ctx)
	if err := h.svc.Test(ctx, user, r.PathValue("provider"), h.availability(ctx)); err != nil {
		writeAIError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (h *AIHandler) Select(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var body struct {
		Source   string `json:"source"`
		Provider string `json:"provider"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	user, _ := auth.UserIDFromContext(r.Context())
	if err := h.svc.Select(r.Context(), user, body.Source, body.Provider, h.availability(r.Context())); err != nil {
		writeAIError(w, err)
		return
	}
	w.WriteHeader(204)
}
