package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
