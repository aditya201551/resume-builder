package repository_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"resume-builder/backend/internal/api"
	"resume-builder/backend/internal/api/handlers"
	"resume-builder/backend/internal/auth"
	"resume-builder/backend/internal/jobmap"
	"resume-builder/backend/internal/repository"
	"resume-builder/backend/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Uses a disposable schema, never the application's users or job tables.
// The local .env is read only in an explicitly requested integration run.
func TestJobMapPostgresIntegration(t *testing.T) {
	if os.Getenv("JOB_MAP_DB_TEST") != "1" {
		t.Skip("set JOB_MAP_DB_TEST=1 to test the migration and API in a disposable Postgres schema")
	}
	dsn := os.Getenv("JOB_MAP_TEST_DATABASE_URL")
	if dsn == "" {
		values, err := godotenv.Read("../../.env")
		if err != nil {
			t.Fatal("provide JOB_MAP_TEST_DATABASE_URL or a local backend/.env")
		}
		dsn = values["DATABASE_URL"]
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("could not initialize integration database")
	}
	defer admin.Close()
	schema := fmt.Sprintf("job_map_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal("could not create isolated test schema")
	}
	defer func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error("could not clean up isolated schema")
		}
	}()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid integration database configuration")
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `CREATE TABLE users(id UUID PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../db/migrations/000006_job_map.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	a, b := "00000000-0000-4000-8000-000000000001", "00000000-0000-4000-8000-000000000002"
	if _, err := pool.Exec(ctx, `INSERT INTO users(id) VALUES($1),($2)`, a, b); err != nil {
		t.Fatal(err)
	}
	repo := repository.NewJobRepository(pool)
	job, ok := jobmap.Prepare(jobmap.Job{ProviderID: "one", Source: "ashby:example", SourceName: "Example careers", SourceKind: "company", Company: "Example", Title: "Software Engineer", ApplyURL: "https://example.com/job/one", SourceURL: "https://example.com/job/one", Workplace: "hybrid", Locations: []jobmap.Location{jobmap.ResolveLocation("Berlin", "DE")}}, time.Now())
	if !ok {
		t.Fatal("fixture invalid")
	}
	if err := repo.StoreSource(ctx, job.Source, []jobmap.Job{job}, true); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, a, job.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, a, job.ID); err != nil {
		t.Fatal("save must be idempotent", err)
	}
	if jobs, err := repo.List(ctx, b, true); err != nil || len(jobs) != 0 {
		t.Fatalf("saved job crossed user boundary: %v %+v", err, jobs)
	}
	if err := repo.Unsave(ctx, b, job.ID); err != nil {
		t.Fatal(err)
	}
	if jobs, err := repo.List(ctx, a, true); err != nil || len(jobs) != 1 || !jobs[0].Saved {
		t.Fatal("another user removed a bookmark", err)
	}
	var wg sync.WaitGroup
	reserved := make(chan bool, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := repo.ReserveRequest(ctx, "test", "2026-10", 3)
			reserved <- ok && err == nil
		}()
	}
	wg.Wait()
	close(reserved)
	count := 0
	for ok := range reserved {
		if ok {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("concurrent budget cap granted %d instead of 3", count)
	}
	if ok, err := repo.ClaimSearch(ctx, "same-search", time.Hour); err != nil || !ok {
		t.Fatal("initial lease", err)
	}
	if ok, err := repo.ClaimSearch(ctx, "same-search", time.Hour); err != nil || ok {
		t.Fatal("duplicate search lease", err)
	}
	jobCfg := jobmap.Config{Companies: []jobmap.Company{{Name: "Example", Website: "https://example.com", Provider: "ashby", Board: "example"}}, RefreshInterval: time.Hour}
	svc := service.NewJobService(repo, jobCfg)
	issuer := auth.NewJWTIssuer("integration-only-secret", time.Hour)
	router := api.NewRouter(issuer, api.Handlers{Jobs: handlers.NewJobHandler(svc)}, func(http.ResponseWriter, *http.Request) {}, "")
	request := func(method, path, user string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		if user != "" {
			token, _ := issuer.Issue(user)
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	if w := request("GET", "/api/jobs", ""); w.Code != 401 {
		t.Fatal("jobs endpoint must require sign-in", w.Code)
	}
	if w := request("GET", "/api/jobs?saved=true", a); w.Code != 200 || !strings.Contains(w.Body.String(), job.ID) {
		t.Fatal("saved jobs API", w.Code, w.Body.String())
	}
	if w := request("GET", "/api/jobs?saved=true", b); w.Code != 200 || strings.Contains(w.Body.String(), job.ID) {
		t.Fatal("saved jobs API user isolation", w.Code)
	}
	if w := request("GET", "/api/jobs?bounds=NaN,0,1,2", a); w.Code != 400 {
		t.Fatal("invalid bounds accepted", w.Code)
	}
	if w := request("GET", "/api/jobs/places?q=Berlin", a); w.Code != 200 || !strings.Contains(w.Body.String(), "Germany") {
		t.Fatal("offline location search", w.Code)
	}
	if err := repo.StoreSource(ctx, job.Source, []jobmap.Job{}, true); err != nil {
		t.Fatal(err)
	}
	if jobs, err := repo.List(ctx, a, false); err != nil || len(jobs) != 0 {
		t.Fatal("closed job remained in discovery", err)
	}
	if jobs, err := repo.List(ctx, a, true); err != nil || len(jobs) != 1 || !jobs[0].Closed {
		t.Fatal("closed bookmark was lost", err)
	}
	if err := repo.Prune(ctx); err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../db/migrations/000006_job_map.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatal("down migration", err)
	}
}
