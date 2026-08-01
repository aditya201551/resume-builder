package repository

import "fmt"

// onConflictClientID returns the ON CONFLICT clause every idempotent create
// in this package uses. Every create request from the frontend carries the
// tempId it generated locally as client_id (see migration
// 000002_client_id_idempotency and frontend/src/lib/tempId.ts) — if the
// browser dies mid-flush after the row was already created but before the
// client durably recorded the resolved id, the retry that follows re-sends
// the same client_id. Instead of inserting a duplicate row, this makes that
// retry a no-op that returns the original row: DO UPDATE SET client_id =
// EXCLUDED.client_id (setting client_id to itself) touches no other column
// but still lets RETURNING produce a row, which a plain DO NOTHING can't
// (Postgres never runs RETURNING on a skipped insert).
//
// scopeColumn is whichever column client_id's uniqueness is scoped to for
// that table — resume_id for top-level entities, or the immediate parent
// column for skill_items/custom_section_entries, matching the partial
// unique index the migration created for that table exactly (both the
// column list and the WHERE clause must match the index for Postgres to
// recognize it as the conflict target).
func onConflictClientID(scopeColumn string) string {
	return fmt.Sprintf(
		"ON CONFLICT (%s, client_id) WHERE client_id IS NOT NULL DO UPDATE SET client_id = EXCLUDED.client_id",
		scopeColumn,
	)
}
