package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func reorder(ctx context.Context, pool *pgxpool.Pool, table, parentColumn, parentID string, orderedIDs []string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	stmt := fmt.Sprintf(`UPDATE %s SET sort_order = $1 WHERE id = $2 AND %s = $3`, table, parentColumn)
	for i, id := range orderedIDs {
		if _, err := tx.Exec(ctx, stmt, i, id, parentID); err != nil {
			return fmt.Errorf("reorder %s: %w", table, err)
		}
	}
	return tx.Commit(ctx)
}
