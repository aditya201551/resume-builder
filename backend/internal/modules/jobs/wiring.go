package jobs

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"resume-builder/backend/internal/ai"
	"resume-builder/backend/internal/modules/jobs/internal/handlers"
	"resume-builder/backend/internal/modules/jobs/internal/jobagent"
	"resume-builder/backend/internal/modules/jobs/internal/jobmap"
	"resume-builder/backend/internal/modules/jobs/internal/repository"
	"resume-builder/backend/internal/modules/jobs/internal/service"
)

func buildHandler(ctx context.Context, pool *pgxpool.Pool, aiConfig *ai.Config) (http.Handler, error) {
	config, err := jobmap.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("load job search config: %w", err)
	}
	repo := repository.NewJobRepository(pool)
	jobs := service.NewJobService(repo, config)
	planner, err := jobagent.New(ctx, aiConfig)
	if err != nil {
		return nil, fmt.Errorf("create job assistant: %w", err)
	}
	handler := handlers.NewJobHandler(jobs).WithDiscovery(
		service.NewJobDiscoveryService(repo, jobs),
		service.NewJobAssistantService(planner, repo),
	)
	mux := http.NewServeMux()
	handler.RegisterRoutes(func(pattern string, handler http.HandlerFunc) { mux.HandleFunc(pattern, handler) })
	return mux, nil
}
