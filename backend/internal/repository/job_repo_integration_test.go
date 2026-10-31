package repository_test

import (
	"context"
	"encoding/json"
	"errors"
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

	"github.com/jackc/pgx/v5/pgconn"
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
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) {
			t.Fatalf("could not create isolated test schema (SQLSTATE %s)", pgError.Code)
		}
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
	legacy := jobmap.Job{ID: "legacy", ProviderID: "legacy", Source: "himalayas", PostedAt: timePtr(time.Now().Add(-time.Hour)), FetchedAt: time.Now()}
	legacyJSON, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO job_listings(id,source,provider_id,payload,fetched_at) VALUES('legacy','himalayas','legacy',$1,now())`, legacyJSON); err != nil {
		t.Fatal(err)
	}
	feedMigration, err := os.ReadFile("../db/migrations/000007_himalayas_feed.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(feedMigration)); err != nil {
		t.Fatal(err)
	}
	externalMigration, err := os.ReadFile("../db/migrations/000008_external_job_search_cache.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(externalMigration)); err != nil {
		t.Fatal(err)
	}
	var migrated []byte
	if err := pool.QueryRow(ctx, `SELECT payload FROM job_listings WHERE id='legacy'`).Scan(&migrated); err != nil {
		t.Fatal(err)
	}
	var converted jobmap.Job
	if err := json.Unmarshal(migrated, &converted); err != nil || converted.PostedAt != nil || converted.SourceUpdatedAt == nil {
		t.Fatal("legacy Himalayas date migration", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM job_listings WHERE id='legacy'`); err != nil {
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
	t.Run("resumable feed and authoritative closures", func(t *testing.T) {
		remote, ok := jobmap.Prepare(jobmap.Job{Source: "himalayas", ProviderID: "https://himalayas.app/companies/acme/jobs/a", Company: "Acme", Title: "Software Engineer", ApplyURL: "https://example.com/a", Workplace: "remote"}, time.Now())
		if !ok {
			t.Fatal("invalid remote fixture")
		}
		progress := jobmap.FeedProgress{Cursor: "opaque-next", JobCount: 1}
		if err := repo.StoreFeedPage(ctx, "himalayas", []jobmap.Job{remote}, progress); err != nil {
			t.Fatal(err)
		}
		if got, err := repo.FeedProgress(ctx, "himalayas"); err != nil || got.Cursor != "opaque-next" || got.JobCount != 1 {
			t.Fatal("checkpoint lost", got, err)
		}
		if err := repo.Save(ctx, a, remote.ID); err != nil {
			t.Fatal(err)
		}
		checked := time.Now().UTC()
		if err := repo.MarkJobClosed(ctx, remote.ID, checked); err != nil {
			t.Fatal(err)
		}
		remote.FetchedAt = time.Now()
		progress.Cursor = ""
		progress.CompletedAt = timePtr(time.Now())
		if err := repo.StoreFeedPage(ctx, "himalayas", []jobmap.Job{remote}, progress); err != nil {
			t.Fatal(err)
		}
		got, err := repo.Get(ctx, a, remote.ID)
		if err != nil || !got.Closed || !got.Saved || got.CheckedAt == nil {
			t.Fatal("cached feed reopened closed job or lost bookmark", got, err)
		}
		// A later explicit 200 can reopen it; an older cached 200 cannot override a newer 410.
		remote.CheckedAt = timePtr(checked.Add(time.Second))
		if err := repo.StoreSource(ctx, "himalayas", []jobmap.Job{remote}, false); err != nil {
			t.Fatal(err)
		}
		got, err = repo.Get(ctx, a, remote.ID)
		if err != nil || got.Closed || !got.Saved {
			t.Fatal("explicit check could not reopen job", got, err)
		}
		if err := repo.MarkJobClosed(ctx, remote.ID, checked.Add(2*time.Second)); err != nil {
			t.Fatal(err)
		}
		if err := repo.StoreSource(ctx, "himalayas", []jobmap.Job{remote}, false); err != nil {
			t.Fatal(err)
		}
		got, err = repo.Get(ctx, a, remote.ID)
		if err != nil || !got.Closed {
			t.Fatal("stale check overrode newer closure", got, err)
		}
		// A checkpoint failure rolls back its page's job writes as well.
		remote.ID = "rolled-back"
		remote.ProviderID = "rolled-back"
		progress.JobCount = -1
		if err := repo.StoreFeedPage(ctx, "himalayas", []jobmap.Job{remote}, progress); err == nil {
			t.Fatal("expected checkpoint constraint failure")
		}
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM job_listings WHERE id='rolled-back'`).Scan(&count); err != nil || count != 0 {
			t.Fatal("page did not roll back", count, err)
		}
	})
	t.Run("external search ownership and successful-result cache", func(t *testing.T) {
		claim := func(key, token string) bool {
			got, err := repo.ClaimExternalSearch(ctx, key, token, time.Minute, 24*time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			return got
		}
		if !claim("q", "first") || claim("q", "second") {
			t.Fatal("concurrent ownership was not exclusive")
		}
		if _, err := pool.Exec(ctx, `UPDATE job_external_search_cache SET lease_until=now()-interval '1 second' WHERE cache_key='q'`); err != nil {
			t.Fatal(err)
		}
		if !claim("q", "new-owner") {
			t.Fatal("expired in-flight lock did not recover")
		}
		if err := repo.ReleaseExternalSearch(ctx, "q", "first"); err != nil {
			t.Fatal(err)
		}
		if claim("q", "third") {
			t.Fatal("old cleanup removed a new owner's lease")
		}
		if err := repo.CompleteExternalSearch(ctx, "q", "first"); err == nil {
			t.Fatal("old owner could complete someone else's search")
		}
		if err := repo.CompleteExternalSearch(ctx, "q", "new-owner"); err != nil {
			t.Fatal(err)
		}
		if claim("q", "cached") {
			t.Fatal("successful result was not cached")
		}
		if _, err := pool.Exec(ctx, `UPDATE job_external_search_cache SET refreshed_at=now()-interval '25 hours' WHERE cache_key='q'`); err != nil {
			t.Fatal(err)
		}
		if !claim("q", "daily-refresh") {
			t.Fatal("successful cache did not expire")
		}
		if err := repo.ReleaseExternalSearch(ctx, "q", "daily-refresh"); err != nil {
			t.Fatal(err)
		}
		if !claim("q", "retry") {
			t.Fatal("failed or interrupted query blocked its retry")
		}
		if _, err := pool.Exec(ctx, `INSERT INTO job_search_cache(cache_key,fetched_at) VALUES('legacy-poison',now())`); err != nil {
			t.Fatal(err)
		}
		if !claim("legacy-poison", "recovered") {
			t.Fatal("legacy canceled query poisoned new cache")
		}
		if !claim("backoff", "failed-owner") {
			t.Fatal("failed-query fixture not claimed")
		}
		if err := repo.FailExternalSearch(ctx, "backoff", "failed-owner", time.Now().Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
		if claim("backoff", "premature-retry") {
			t.Fatal("provider failure caused an immediate retry")
		}
		if _, err := pool.Exec(ctx, `UPDATE job_external_search_cache SET retry_at=now()-interval '1 second' WHERE cache_key='backoff'`); err != nil {
			t.Fatal(err)
		}
		if !claim("backoff", "recovered-owner") {
			t.Fatal("failed-query backoff did not expire")
		}
		results := make(chan bool, 20)
		var claims sync.WaitGroup
		for i := 0; i < 20; i++ {
			claims.Add(1)
			go func(i int) {
				defer claims.Done()
				owned, err := repo.ClaimExternalSearch(ctx, "concurrent", fmt.Sprintf("owner-%d", i), time.Minute, 24*time.Hour)
				if err != nil {
					t.Error(err)
				}
				results <- owned
			}(i)
		}
		claims.Wait()
		close(results)
		owners := 0
		for owned := range results {
			if owned {
				owners++
			}
		}
		if owners != 1 {
			t.Fatal("concurrent provider requests were not deduplicated", owners)
		}
	})
	externalDown, err := os.ReadFile("../db/migrations/000008_external_job_search_cache.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(externalDown)); err != nil {
		t.Fatal("external search down migration", err)
	}
	feedDown, err := os.ReadFile("../db/migrations/000007_himalayas_feed.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(feedDown)); err != nil {
		t.Fatal("feed down migration", err)
	}
	down, err := os.ReadFile("../db/migrations/000006_job_map.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatal("down migration", err)
	}
}

func timePtr(t time.Time) *time.Time { return &t }
