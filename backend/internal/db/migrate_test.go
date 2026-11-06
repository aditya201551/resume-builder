package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"os"
	"testing"
	"time"
)

// Explicit opt-in: exercise the real startup migrator in a disposable schema.
func TestBaselineMigrationLifecycle(t *testing.T) {
	dsn := os.Getenv("MIGRATIONS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set MIGRATIONS_TEST_DATABASE_URL for PostgreSQL migration integration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("baseline_test_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	token := stdlib.RegisterConnConfig(cfg.ConnConfig)
	defer stdlib.UnregisterConnConfig(token)
	if err = RunMigrations(token); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var version int
	var dirty bool
	if err = pool.QueryRow(ctx, "SELECT version,dirty FROM schema_migrations").Scan(&version, &dirty); err != nil || version < 1 || dirty {
		t.Fatal("startup migration state is invalid", err)
	}
	var retired int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema=$1 AND (table_name='templates' OR table_name LIKE 'job_%' OR table_name='saved_jobs')`, schema).Scan(&retired); err != nil || retired != 0 {
		t.Fatal("retired schema remains", err)
	}
	var users int
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&users); err != nil || users != 0 {
		t.Fatal("baseline includes test users", err)
	}
	if _, err = pool.Exec(ctx, "INSERT INTO users(email) VALUES('migration-test@example.invalid')"); err != nil {
		t.Fatal(err)
	}
	if err = RunMigrations(token); err != nil {
		t.Fatal("repeated startup failed", err)
	}
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&users); err != nil || users != 1 {
		t.Fatal("repeated startup changed existing data", err)
	}
}
