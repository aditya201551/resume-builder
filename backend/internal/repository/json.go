package repository

import "encoding/json"

// defaultJSON substitutes fallback when raw is nil (the client omitted the
// field), so a NOT NULL DEFAULT '...' jsonb column gets its declared default
// instead of an explicit SQL NULL — passing json.RawMessage(nil) as a query
// arg sends NULL, which bypasses the column's DEFAULT entirely.
func defaultJSON(raw json.RawMessage, fallback string) json.RawMessage {
	if raw == nil {
		return json.RawMessage(fallback)
	}
	return raw
}
