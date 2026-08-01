package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ResumeDesign is the raw stored row — internal/design.ResumeDesign is the
// typed/validated shape; this package only shuttles the JSONB blob, it
// never inspects it (same separation as Resume.Links).
type ResumeDesign struct {
	ResumeID string          `json:"resume_id"`
	Design   json.RawMessage `json:"design"`
}

type ResumeDesignRepository struct {
	pool *pgxpool.Pool
}

func NewResumeDesignRepository(pool *pgxpool.Pool) *ResumeDesignRepository {
	return &ResumeDesignRepository{pool: pool}
}

// Find returns ErrNotFound when the resume has never had a design saved —
// callers fall back to the default template's preset in that case rather
// than treating it as an error.
func (r *ResumeDesignRepository) Find(ctx context.Context, resumeID string) (*ResumeDesign, error) {
	const q = `SELECT resume_id, design FROM resume_designs WHERE resume_id = $1`
	var d ResumeDesign
	err := r.pool.QueryRow(ctx, q, resumeID).Scan(&d.ResumeID, &d.Design)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find resume design: %w", err)
	}
	return &d, nil
}

// Upsert is keyed by resume_id (the table's primary key) rather than a
// synthetic row id, since a resume has at most one design — same pattern
// as SectionConfigRepository.Update's upsert, simpler here because there's
// no composite identity to branch on.
func (r *ResumeDesignRepository) Upsert(ctx context.Context, resumeID string, design json.RawMessage) (*ResumeDesign, error) {
	const q = `
		INSERT INTO resume_designs (resume_id, design)
		VALUES ($1, $2)
		ON CONFLICT (resume_id) DO UPDATE SET design = $2, updated_at = now()
		RETURNING resume_id, design`
	var d ResumeDesign
	err := r.pool.QueryRow(ctx, q, resumeID, design).Scan(&d.ResumeID, &d.Design)
	if err != nil {
		return nil, fmt.Errorf("upsert resume design: %w", err)
	}
	return &d, nil
}
