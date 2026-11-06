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
		sql, err := os.ReadFile("../../db/migrations/000010_feature_flags." + name + ".sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatal(err)
		}
	}
	apply("up")
	if _, err := pool.Exec(ctx, `INSERT INTO feature_flags(key, enabled, updated_by) VALUES ('test.enabled', false, 'integration')`); err != nil {
		t.Fatal(err)
	}
	store := postgres.NewStore(pool)
	if err := store.Set(ctx, "test.enabled", true, "integration-owner"); err != nil {
		t.Fatal(err)
	}
	stored, err := store.List(ctx)
	if err != nil || !stored["test.enabled"] {
		t.Fatal("update not visible", err)
	}
	var actor string
	if err := pool.QueryRow(ctx, `SELECT actor FROM feature_flag_audit ORDER BY id DESC LIMIT 1`).Scan(&actor); err != nil || actor != "integration-owner" {
		t.Fatal("administrative update was not audited", err)
	}
	// A rejected audit insert must roll back its flag update as well.
	if _, err := pool.Exec(ctx, `ALTER TABLE feature_flag_audit ADD CONSTRAINT reject_fixture CHECK(actor <> 'reject')`); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(ctx, "test.enabled", false, "reject"); err == nil {
		t.Fatal("audit failure was ignored")
	}
	stored, err = store.List(ctx)
	if err != nil || !stored["test.enabled"] {
		t.Fatal("flag update survived an audit failure", err)
	}
	if err := store.Set(ctx, "test.enabled", false, "integration-owner"); err != nil {
		t.Fatal(err)
	}
	stored, err = store.List(ctx)
	if err != nil || stored["test.enabled"] {
		t.Fatal("disable not visible", err)
	}
	var changes int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM feature_flag_audit WHERE feature_key = 'test.enabled'`).Scan(&changes); err != nil || changes != 2 {
		t.Fatal("unexpected audit history", changes, err)
	}
	apply("down")
	if _, err := store.List(ctx); err == nil {
		t.Fatal("down migration did not remove registry")
	}
}
