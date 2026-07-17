package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// reorder sets sort_order = position for each id in orderedIDs, scoped to
// parentColumn = parentID, in one transaction. table/parentColumn are always
// internal string literals supplied by callers in this package, never
// request data, so building the statement with fmt.Sprintf is safe here.
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
