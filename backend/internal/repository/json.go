package repository

import "encoding/json"

func defaultJSON(raw json.RawMessage, fallback string) json.RawMessage {
	if raw == nil {
		return json.RawMessage(fallback)
	}
	return raw
}
