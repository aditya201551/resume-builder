package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Template struct {
	ID              string          `json:"id"`
	RendererKey     string          `json:"renderer_key"`
	Name            string          `json:"name"`
	ThumbnailURL    *string         `json:"thumbnail_url"`
	Tags            json.RawMessage `json:"tags"`
	SupportedModes  json.RawMessage `json:"supported_modes"`
	SupportedGroups json.RawMessage `json:"supported_groups"`
	DefaultDesign   json.RawMessage `json:"default_design"`
	IsPremium       bool            `json:"is_premium"`
	Published       bool            `json:"published"`
	SortOrder       int             `json:"sort_order"`
}

type TemplateRepository struct {
	pool *pgxpool.Pool
}

func NewTemplateRepository(pool *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{pool: pool}
}

const templateColumns = `id, renderer_key, name, thumbnail_url, tags, supported_modes, supported_groups, default_design, is_premium, published, sort_order`

func scanTemplate(row pgx.Row) (*Template, error) {
	var t Template
	err := row.Scan(&t.ID, &t.RendererKey, &t.Name, &t.ThumbnailURL, &t.Tags,
		&t.SupportedModes, &t.SupportedGroups, &t.DefaultDesign, &t.IsPremium, &t.Published, &t.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TemplateRepository) List(ctx context.Context) ([]Template, error) {
	const q = `SELECT ` + templateColumns + ` FROM templates WHERE published = true ORDER BY sort_order`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	templates := []Template{}
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, *t)
	}
	return templates, rows.Err()
}

func (r *TemplateRepository) FindByID(ctx context.Context, id string) (*Template, error) {
	const q = `SELECT ` + templateColumns + ` FROM templates WHERE id = $1`
	return scanTemplate(r.pool.QueryRow(ctx, q, id))
}

func (r *TemplateRepository) FindByRendererKey(ctx context.Context, rendererKey string) (*Template, error) {
	const q = `SELECT ` + templateColumns + ` FROM templates WHERE renderer_key = $1`
	return scanTemplate(r.pool.QueryRow(ctx, q, rendererKey))
}
