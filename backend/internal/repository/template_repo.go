package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Template struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	Slug                string          `json:"slug"`
	Regions             json.RawMessage `json:"regions"`
	DefaultRegionMap    json.RawMessage `json:"default_region_map"`
	PreviewThumbnailURL *string         `json:"preview_thumbnail_url"`
	IsActive            bool            `json:"is_active"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type TemplateRepository struct {
	pool *pgxpool.Pool
}

func NewTemplateRepository(pool *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{pool: pool}
}

const templateColumns = `id, name, slug, regions, default_region_map, preview_thumbnail_url, is_active, created_at, updated_at`

func (r *TemplateRepository) ListActive(ctx context.Context) ([]Template, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+templateColumns+` FROM templates WHERE is_active = true ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	items := []Template{}
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Regions, &t.DefaultRegionMap, &t.PreviewThumbnailURL, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

func (r *TemplateRepository) FindByID(ctx context.Context, id string) (*Template, error) {
	var t Template
	err := r.pool.QueryRow(ctx, `SELECT `+templateColumns+` FROM templates WHERE id = $1`, id).
		Scan(&t.ID, &t.Name, &t.Slug, &t.Regions, &t.DefaultRegionMap, &t.PreviewThumbnailURL, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find template: %w", err)
	}
	return &t, nil
}
