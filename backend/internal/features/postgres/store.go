package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
)

type database interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
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
