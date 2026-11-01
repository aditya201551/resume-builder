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
	"resume-builder/backend/internal/jobsearch"
	"resume-builder/backend/internal/repository"
	"resume-builder/backend/internal/service"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Uses a disposable schema, never the application's users or job tables.
// The local .env is read only in an explicitly requested integration run.
func TestJobDiscoveryPostgresIntegration(t *testing.T) {
	if os.Getenv("JOB_SEARCH_DB_TEST") != "1" {
		t.Skip("set JOB_SEARCH_DB_TEST=1 to test the migration and API in a disposable Postgres schema")
	}
	dsn := os.Getenv("JOB_SEARCH_TEST_DATABASE_URL")
	if dsn == "" {
		values, err := godotenv.Read("../../.env")
		if err != nil {
			t.Fatal("provide JOB_SEARCH_TEST_DATABASE_URL or a local backend/.env")
		}
		dsn = values["DATABASE_URL"]
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("could not initialize integration database")
	}
	defer admin.Close()
	schema := fmt.Sprintf("job_search_test_%d", time.Now().UnixNano())
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
	discoveryMigration, err := os.ReadFile("../db/migrations/000009_job_discovery_pages.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(discoveryMigration)); err != nil {
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
	if err := repo.StoreSource(ctx, job.Source, []jobmap.Job{job}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, a, job.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, a, job.ID); err != nil {
		t.Fatal("save must be idempotent", err)
	}
	if jobs, _, err := repo.SavedDiscoveryPage(ctx, b, 0); err != nil || len(jobs) != 0 {
		t.Fatalf("saved job crossed user boundary: %v %+v", err, jobs)
	}
	if err := repo.Unsave(ctx, b, job.ID); err != nil {
		t.Fatal(err)
	}
	if jobs, _, err := repo.SavedDiscoveryPage(ctx, a, 0); err != nil || len(jobs) != 1 || !jobs[0].Saved {
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
	svc := service.NewJobService(repo, jobmap.Config{})
	issuer := auth.NewJWTIssuer("integration-only-secret", time.Hour)
	discovery := service.NewJobDiscoveryService(repo, svc)
	router := api.NewRouter(issuer, api.Handlers{Jobs: handlers.NewJobHandler(svc).WithDiscovery(discovery, nil)}, func(http.ResponseWriter, *http.Request) {}, "")
	request := func(method, path, user, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if user != "" {
			token, _ := issuer.Issue(user)
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	savedRequest := `{"intent":{"version":1,"branches":[{"workplace":"all","country":"","city":""}]},"savedOnly":true}`
	if w := request("POST", "/api/jobs/search", "", savedRequest); w.Code != 401 {
		t.Fatal("discovery requires sign-in", w.Code)
	}
	if w := request("POST", "/api/jobs/search", a, savedRequest); w.Code != 200 || !strings.Contains(w.Body.String(), job.ID) {
		t.Fatal("saved discovery", w.Code, w.Body.String())
	}
	if w := request("POST", "/api/jobs/search", b, savedRequest); w.Code != 200 || strings.Contains(w.Body.String(), job.ID) {
		t.Fatal("saved discovery crossed user boundary", w.Code)
	}
	if w := request("POST", "/api/jobs/search", a, `{"intent":{"version":1,"branches":[{"workplace":"teleport"}]}}`); w.Code != 400 {
		t.Fatal("invalid criteria accepted", w.Code)
	}
	if w := request("GET", "/api/jobs/countries", a, ""); w.Code != 200 || !strings.Contains(w.Body.String(), "Germany") {
		t.Fatal("country list", w.Code)
	}
	if err := repo.MarkJobClosed(ctx, job.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if jobs, _, err := repo.SavedDiscoveryPage(ctx, a, 0); err != nil || len(jobs) != 1 || !jobs[0].Closed {
		t.Fatal("closed bookmark was lost", err)
	}
	t.Run("availability checks preserve authoritative closures", func(t *testing.T) {
		remote, ok := jobmap.Prepare(jobmap.Job{Source: "himalayas", ProviderID: "https://himalayas.app/companies/acme/jobs/a", Company: "Acme", Title: "Software Engineer", ApplyURL: "https://example.com/a", Workplace: "remote"}, time.Now())
		if !ok {
			t.Fatal("invalid remote fixture")
		}
		if err := repo.StoreSource(ctx, "himalayas", []jobmap.Job{remote}); err != nil {
			t.Fatal(err)
		}
		if err := repo.Save(ctx, a, remote.ID); err != nil {
			t.Fatal(err)
		}
		checked := time.Now().UTC()
		if err := repo.MarkJobClosed(ctx, remote.ID, checked); err != nil {
			t.Fatal(err)
		}
		if err := repo.StoreSource(ctx, "himalayas", []jobmap.Job{remote}); err != nil {
			t.Fatal(err)
		}
		if got, err := repo.Get(ctx, a, remote.ID); err != nil || !got.Closed || !got.Saved || got.CheckedAt == nil {
			t.Fatal("cached result reopened closed bookmark", err)
		}
		remote.CheckedAt = timePtr(checked.Add(time.Second))
		if err := repo.StoreSource(ctx, "himalayas", []jobmap.Job{remote}); err != nil {
			t.Fatal(err)
		}
		if got, err := repo.Get(ctx, a, remote.ID); err != nil || got.Closed || !got.Saved {
			t.Fatal("explicit check did not reopen job", err)
		}
		if err := repo.MarkJobClosed(ctx, remote.ID, checked.Add(2*time.Second)); err != nil {
			t.Fatal(err)
		}
		if err := repo.StoreSource(ctx, "himalayas", []jobmap.Job{remote}); err != nil {
			t.Fatal(err)
		}
		if got, err := repo.Get(ctx, a, remote.ID); err != nil || !got.Closed {
			t.Fatal("stale check overrode closure", err)
		}
		if err := repo.DelayFeed(ctx, "himalayas", checked.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		if retryAt, err := repo.ProviderRetryAt(ctx, "himalayas"); err != nil || retryAt == nil || !retryAt.After(checked) {
			t.Fatal("provider backoff was not persisted", err)
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
	t.Run("bounded discovery pages preserve bookmarks and user isolation", func(t *testing.T) {
		makeJob := func(id string) jobmap.Job {
			j, ok := jobmap.Prepare(jobmap.Job{Source: "himalayas", ProviderID: id, Company: "Sample", Title: "Software Engineer", ApplyURL: "https://example.com/" + id, Workplace: "remote", Remote: jobmap.RemotePolicy{Scope: "worldwide"}}, time.Now())
			if !ok {
				t.Fatal("invalid fixture")
			}
			return j
		}
		kept, evicted := makeJob("discovery-kept"), makeJob("discovery-evicted")
		page := jobsearch.Page{Jobs: []jobmap.Job{kept, evicted}, Cursor: "next", HasMore: true, FetchedAt: time.Now()}
		if err := repo.StoreDiscoveryPage(ctx, "old-page", page); err != nil {
			t.Fatal(err)
		}
		if err := repo.Save(ctx, a, kept.ID); err != nil {
			t.Fatal(err)
		}
		for _, user := range []string{a, b} {
			p, found, err := repo.DiscoveryPage(ctx, user, "old-page")
			if err != nil || !found || len(p.Jobs) != 2 || p.Cursor != "next" || p.Jobs[0].ID != kept.ID || p.Jobs[0].Saved != (user == a) {
				t.Fatalf("shared cache leaked private state or order: %+v %v", p, err)
			}
		}
		// Fill the cache in one statement, then the public write path must evict
		// old membership and only delete unbookmarked, unreferenced listings.
		if _, err := pool.Exec(ctx, `UPDATE job_discovery_pages SET fetched_at=now()-interval '23 hours' WHERE cache_key='old-page'`); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO job_discovery_pages(cache_key,job_ids,has_more) SELECT 'page-'||i,ARRAY[]::text[],false FROM generate_series(1,501) i`); err != nil {
			t.Fatal(err)
		}
		if err := repo.StoreDiscoveryPage(ctx, "latest-page", jobsearch.Page{Jobs: []jobmap.Job{}, FetchedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM job_discovery_pages`).Scan(&count); err != nil || count != 500 {
			t.Fatal("cache not bounded", count, err)
		}
		if _, found, err := repo.DiscoveryPage(ctx, a, "old-page"); err != nil || found {
			t.Fatal("evicted page remained readable", err)
		}
		if _, err := repo.Get(ctx, a, evicted.ID); !errors.Is(err, repository.ErrNotFound) {
			t.Fatal("unreferenced cached job was retained", err)
		}
		j, err := repo.Get(ctx, a, kept.ID)
		if err != nil || !j.Saved {
			t.Fatal("eviction deleted a bookmark", err)
		}
		if _, err := repo.Get(ctx, b, kept.ID); err != nil {
			t.Fatal("public listing disappeared", err)
		}
		if jobs, _, err := repo.SavedDiscoveryPage(ctx, b, 0); err != nil || len(jobs) != 0 {
			t.Fatal("saved page crossed user boundary", err)
		}
	})
	discoveryDown, err := os.ReadFile("../db/migrations/000009_job_discovery_pages.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(discoveryDown)); err != nil {
		t.Fatal("discovery down migration", err)
	}
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
