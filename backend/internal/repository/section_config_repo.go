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
	IsVisible            bool    `json:"is_visible"`
	DisplayTitleOverride *string `json:"display_title_override"`
	SortOrder            int     `json:"sort_order"`
}

type SectionConfigInput struct {
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

const sectionConfigColumns = `id, resume_id, section_type, custom_section_id, is_visible, display_title_override, sort_order`

func (r *SectionConfigRepository) List(ctx context.Context, resumeID string) ([]SectionConfig, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+sectionConfigColumns+` FROM resume_section_configs WHERE resume_id = $1 ORDER BY sort_order`, resumeID)
	if err != nil {
		return nil, fmt.Errorf("list section configs: %w", err)
	}
	defer rows.Close()

	items := []SectionConfig{}
	for rows.Next() {
		var c SectionConfig
		if err := rows.Scan(&c.ID, &c.ResumeID, &c.SectionType, &c.CustomSectionID, &c.IsVisible, &c.DisplayTitleOverride, &c.SortOrder); err != nil {
			return nil, fmt.Errorf("scan section config: %w", err)
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

func (r *SectionConfigRepository) Update(ctx context.Context, resumeID, sectionType string, customSectionID *string, in SectionConfigInput) (*SectionConfig, error) {
	q := `
		INSERT INTO resume_section_configs (resume_id, section_type, custom_section_id, is_visible, display_title_override, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (resume_id, section_type, custom_section_id) DO UPDATE SET
			is_visible = $4, display_title_override = $5, sort_order = $6
		RETURNING ` + sectionConfigColumns
	if customSectionID == nil {
		q = `
			INSERT INTO resume_section_configs (resume_id, section_type, custom_section_id, is_visible, display_title_override, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (resume_id, section_type) WHERE custom_section_id IS NULL DO UPDATE SET
				is_visible = $4, display_title_override = $5, sort_order = $6
			RETURNING ` + sectionConfigColumns
	}

	var c SectionConfig
	err := r.pool.QueryRow(ctx, q, resumeID, sectionType, customSectionID, in.IsVisible, in.DisplayTitleOverride, in.SortOrder).
		Scan(&c.ID, &c.ResumeID, &c.SectionType, &c.CustomSectionID, &c.IsVisible, &c.DisplayTitleOverride, &c.SortOrder)
	if err != nil {
		return nil, fmt.Errorf("update section config: %w", err)
	}
	return &c, nil
}

func (r *SectionConfigRepository) EnsureExists(ctx context.Context, resumeID, sectionType string, customSectionID *string, sortOrder int) error {
	const q = `
		INSERT INTO resume_section_configs (resume_id, section_type, custom_section_id, is_visible, sort_order)
		VALUES ($1, $2, $3, true, $4)
		ON CONFLICT (resume_id, section_type, custom_section_id) DO NOTHING`
	if _, err := r.pool.Exec(ctx, q, resumeID, sectionType, customSectionID, sortOrder); err != nil {
		return fmt.Errorf("ensure section config: %w", err)
	}
	return nil
}
