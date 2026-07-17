package service

import "errors"

// ErrNotFound is returned both when a row genuinely doesn't exist and when it
// exists but belongs to a different user — callers must not distinguish the
// two in the HTTP response, or resume IDs become enumerable.
var ErrNotFound = errors.New("not found")

// ErrValidation marks input that's well-formed JSON but violates a business
// rule (e.g. an out-of-enum value) — handlers map this to 400, not 500.
var ErrValidation = errors.New("validation error")
