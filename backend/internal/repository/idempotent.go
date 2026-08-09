package repository

import "fmt"

func onConflictClientID(scopeColumn string) string {
	return fmt.Sprintf(
		"ON CONFLICT (%s, client_id) WHERE client_id IS NOT NULL DO UPDATE SET client_id = EXCLUDED.client_id",
		scopeColumn,
	)
}
