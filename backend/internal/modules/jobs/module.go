// Package jobs is the public boundary of the Jobs module. Its internal packages
// own HTTP adapters, business logic, persistence, providers and search planning.
package jobs

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"resume-builder/backend/internal/ai"
	"resume-builder/backend/internal/api/middleware"
	"resume-builder/backend/internal/features"
	"resume-builder/backend/internal/platform/httpresponse"
)

type Module struct {
	flags   middleware.FeatureEvaluator
	mu      sync.Mutex
	build   func() (http.Handler, error)
	handler http.Handler
	retryAt time.Time
}

// New does no Jobs-specific initialization. A disabled module cannot prevent
// the core application from starting because of its provider configuration.
func New(ctx context.Context, pool *pgxpool.Pool, config *ai.Config, flags middleware.FeatureEvaluator) *Module {
	return &Module{flags: flags, build: func() (http.Handler, error) {
		return buildHandler(ctx, pool, config)
	}}
}

// Routes stay registered so a flag change takes effect without restarting.
// Authentication is applied by the application registrar before this gate.
func (m *Module) RegisterRoutes(register func(string, http.HandlerFunc)) {
	handler := middleware.RequireFeature(m.flags, features.Jobs)(http.HandlerFunc(m.serve))
	register("/api/jobs", handler.ServeHTTP)
	register("/api/jobs/", handler.ServeHTTP)
}

func (m *Module) initialized() http.Handler {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.handler != nil {
		return m.handler
	}
	if time.Now().Before(m.retryAt) {
		return nil
	}
	handler, err := m.build()
	if err != nil {
		m.retryAt = time.Now().Add(10 * time.Second)
		slog.Error("jobs_initialization_failed", "error", err)
		return nil
	}
	m.handler = handler
	return handler
}

func (m *Module) serve(w http.ResponseWriter, r *http.Request) {
	handler := m.initialized()
	if handler == nil {
		httpresponse.JSON(w, http.StatusServiceUnavailable, map[string]string{"code": "module_unavailable", "message": "Jobs is temporarily unavailable."})
		return
	}
	handler.ServeHTTP(w, r)
}
