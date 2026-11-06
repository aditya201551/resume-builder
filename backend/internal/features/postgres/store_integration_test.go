package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"resume-builder/backend/internal/features/postgres"
)

// Set FEATURE_FLAGS_TEST_DATABASE_URL to run against a disposable schema.
// No application environment file or production credentials are read.
func TestFeatureFlagsPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("FEATURE_FLAGS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set FEATURE_FLAGS_TEST_DATABASE_URL for PostgreSQL integration")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("feature_flags_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	apply := func(name string) {
		t.Helper()
		sql, err := os.ReadFile("../../db/migrations/000001_init_schema." + name + ".sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatal(err)
		}
	}
	apply("up")
	var retiredTables int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema=$1 AND (table_name='templates' OR table_name LIKE 'job_%' OR table_name='saved_jobs')`, schema).Scan(&retiredTables); err != nil || retiredTables != 0 {
		t.Fatal("baseline retained retired tables", err)
	}
	initial, err := postgres.NewStore(pool).List(ctx)
	if err != nil || len(initial) != 3 || initial["platform_ai"] || initial["platform_ai_public"] || initial["byok_ai"] {
		t.Fatal("unexpected baseline feature defaults", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO feature_flags(key, enabled, updated_by) VALUES ('test.enabled', false, 'integration')`); err != nil {
		t.Fatal(err)
	}
	store := postgres.NewStore(pool)
	stored, err := store.List(ctx)
	if err != nil || stored["test.enabled"] {
		t.Fatal("initial disabled value not visible", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE feature_flags SET enabled = true WHERE key = 'test.enabled'`); err != nil {
		t.Fatal(err)
	}
	stored, err = store.List(ctx)
	if err != nil || !stored["test.enabled"] {
		t.Fatal("update not visible", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE feature_flags SET enabled = false WHERE key = 'test.enabled'`); err != nil {
		t.Fatal(err)
	}
	stored, err = store.List(ctx)
	if err != nil || stored["test.enabled"] {
		t.Fatal("disable not visible", err)
	}
	apply("down")
	if _, err := store.List(ctx); err == nil {
		t.Fatal("down migration did not remove registry")
	}
	apply("up")
	if _, err := store.List(ctx); err != nil {
		t.Fatal("baseline could not be reapplied", err)
	}
}
