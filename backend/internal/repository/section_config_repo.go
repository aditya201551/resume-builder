package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SectionConfig struct {
	ID                   string  `json:"id"`
	ResumeID             string  `json:"resume_id"`
	SectionType          string  `json:"section_type"`
	CustomSectionID      *string `json:"custom_section_id"`
	Region               *string `json:"region"`
	IsVisible            bool    `json:"is_visible"`
	DisplayTitleOverride *string `json:"display_title_override"`
	SortOrder            int     `json:"sort_order"`
}

type SectionConfigInput struct {
	Region               *string `json:"region"`
	IsVisible            bool    `json:"is_visible"`
	DisplayTitleOverride *string `json:"display_title_override"`
	SortOrder            int     `json:"sort_order"`
}

type SectionConfigRepository struct {
	pool *pgxpool.Pool
}

func NewSectionConfigRepository(pool *pgxpool.Pool) *SectionConfigRepository {
	return &SectionConfigRepository{pool: pool}
}

const sectionConfigColumns = `id, resume_id, section_type, custom_section_id, region, is_visible, display_title_override, sort_order`

func (r *SectionConfigRepository) List(ctx context.Context, resumeID string) ([]SectionConfig, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+sectionConfigColumns+` FROM resume_section_configs WHERE resume_id = $1 ORDER BY sort_order`, resumeID)
	if err != nil {
		return nil, fmt.Errorf("list section configs: %w", err)
	}
	defer rows.Close()

	items := []SectionConfig{}
	for rows.Next() {
		var c SectionConfig
		if err := rows.Scan(&c.ID, &c.ResumeID, &c.SectionType, &c.CustomSectionID, &c.Region, &c.IsVisible, &c.DisplayTitleOverride, &c.SortOrder); err != nil {
			return nil, fmt.Errorf("scan section config: %w", err)
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

// Update upserts by (resume_id, section_type, custom_section_id) rather than
// a row id — that triple is the config's real identity (see the schema's
// UNIQUE constraint), and it's what a client naturally has in hand after
// listing sections rather than a synthetic config row id. It upserts (not a
// plain UPDATE) because resume_section_configs is only ever seeded for
// section types a resume happened to already have data in — a section type
// that's never been touched before (e.g. reordering a section that's never
// been dragged) has no row yet, and a plain UPDATE would silently affect 0
// rows instead of persisting the change.
func (r *SectionConfigRepository) Update(ctx context.Context, resumeID, sectionType string, customSectionID *string, in SectionConfigInput) (*SectionConfig, error) {
	// Two conflict targets because a plain UNIQUE constraint treats NULL
	// custom_section_id as always distinct — ON CONFLICT can't target it for
	// the non-custom case, hence the partial unique index + branch here.
	q := `
		INSERT INTO resume_section_configs (resume_id, section_type, custom_section_id, region, is_visible, display_title_override, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (resume_id, section_type, custom_section_id) DO UPDATE SET
			region = $4, is_visible = $5, display_title_override = $6, sort_order = $7
		RETURNING ` + sectionConfigColumns
	if customSectionID == nil {
		q = `
			INSERT INTO resume_section_configs (resume_id, section_type, custom_section_id, region, is_visible, display_title_override, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (resume_id, section_type) WHERE custom_section_id IS NULL DO UPDATE SET
				region = $4, is_visible = $5, display_title_override = $6, sort_order = $7
			RETURNING ` + sectionConfigColumns
	}

	var c SectionConfig
	err := r.pool.QueryRow(ctx, q, resumeID, sectionType, customSectionID, in.Region, in.IsVisible, in.DisplayTitleOverride, in.SortOrder).
		Scan(&c.ID, &c.ResumeID, &c.SectionType, &c.CustomSectionID, &c.Region, &c.IsVisible, &c.DisplayTitleOverride, &c.SortOrder)
	if err != nil {
		return nil, fmt.Errorf("update section config: %w", err)
	}
	return &c, nil
}

// EnsureExists creates a default config row for a section_type if one
// doesn't already exist — used the first time a custom section is added,
// since resume_section_configs is normally seeded only at resume/template
// creation time and custom sections are created afterwards.
func (r *SectionConfigRepository) EnsureExists(ctx context.Context, resumeID, sectionType string, customSectionID *string, region string, sortOrder int) error {
	const q = `
		INSERT INTO resume_section_configs (resume_id, section_type, custom_section_id, region, is_visible, sort_order)
		VALUES ($1, $2, $3, $4, true, $5)
		ON CONFLICT (resume_id, section_type, custom_section_id) DO NOTHING`
	if _, err := r.pool.Exec(ctx, q, resumeID, sectionType, customSectionID, region, sortOrder); err != nil {
		return fmt.Errorf("ensure section config: %w", err)
	}
	return nil
}
