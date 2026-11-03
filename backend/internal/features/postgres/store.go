package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("feature flag not found")

type database interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Store struct{ db database }

func NewStore(db database) *Store { return &Store{db: db} }

func (r *Store) List(ctx context.Context) (map[string]bool, error) {
	rows, err := r.db.Query(ctx, `SELECT key, enabled FROM feature_flags`)
	if err != nil {
		return nil, fmt.Errorf("list feature flags: %w", err)
	}
	defer rows.Close()
	values := make(map[string]bool)
	for rows.Next() {
		var key string
		var enabled bool
		if err := rows.Scan(&key, &enabled); err != nil {
			return nil, err
		}
		values[key] = enabled
	}
	return values, rows.Err()
}

func (r *Store) Set(ctx context.Context, key string, enabled bool, actor string) error {
	// The flag and its audit record change in a single database statement.
	result, err := r.db.Exec(ctx, `
		WITH changed AS (
			UPDATE feature_flags SET enabled = $2, updated_by = $3, updated_at = now()
			WHERE key = $1 RETURNING key, enabled, updated_by, updated_at
		)
		INSERT INTO feature_flag_audit(feature_key, enabled, actor, changed_at)
		SELECT key, enabled, updated_by, updated_at FROM changed`, key, enabled, actor)
	if err != nil {
		return fmt.Errorf("set feature flag: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
