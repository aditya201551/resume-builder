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

type Project struct {
	ID           string          `json:"id"`
	ResumeID     string          `json:"resume_id"`
	Name         string          `json:"name"`
	Content      string          `json:"content"`
	Role         *string         `json:"role"`
	Technologies json.RawMessage `json:"technologies"`
	URL          *string         `json:"url"`
	StartDate    *time.Time      `json:"start_date"`
	EndDate      *time.Time      `json:"end_date"`
	SortOrder    int             `json:"sort_order"`
}

type ProjectInput struct {
	Name         string          `json:"name"`
	Content      string          `json:"content"`
	Role         *string         `json:"role"`
	Technologies json.RawMessage `json:"technologies"`
	URL          *string         `json:"url"`
	StartDate    *time.Time      `json:"start_date"`
	EndDate      *time.Time      `json:"end_date"`
	SortOrder    int             `json:"sort_order"`
	ClientID     *string         `json:"client_id"`
}

type ProjectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

const projectColumns = `id, resume_id, name, content, role, technologies, url, start_date, end_date, sort_order`

func scanProject(row pgx.Row) (*Project, error) {
	var p Project
	err := row.Scan(&p.ID, &p.ResumeID, &p.Name, &p.Content, &p.Role, &p.Technologies, &p.URL, &p.StartDate, &p.EndDate, &p.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepository) List(ctx context.Context, resumeID string) ([]Project, error) {
	const q = `SELECT ` + projectColumns + ` FROM projects WHERE resume_id = $1 ORDER BY sort_order`
	rows, err := r.pool.Query(ctx, q, resumeID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	items := []Project{}
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		items = append(items, *p)
	}
	return items, rows.Err()
}

func (r *ProjectRepository) Create(ctx context.Context, resumeID string, in ProjectInput) (*Project, error) {
	q := `
		INSERT INTO projects (resume_id, name, content, role, technologies, url, start_date, end_date, sort_order, client_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		` + onConflictClientID("resume_id") + `
		RETURNING ` + projectColumns

	p, err := scanProject(r.pool.QueryRow(ctx, q, resumeID, in.Name, in.Content, in.Role, defaultJSON(in.Technologies, `[]`), in.URL, in.StartDate, in.EndDate, in.SortOrder, in.ClientID))
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return p, nil
}

func (r *ProjectRepository) Update(ctx context.Context, resumeID, id string, in ProjectInput) (*Project, error) {
	const q = `
		UPDATE projects SET
			name = $3, content = $4, role = $5, technologies = $6, url = $7, start_date = $8, end_date = $9, sort_order = $10
		WHERE id = $1 AND resume_id = $2
		RETURNING ` + projectColumns

	p, err := scanProject(r.pool.QueryRow(ctx, q, id, resumeID, in.Name, in.Content, in.Role, defaultJSON(in.Technologies, `[]`), in.URL, in.StartDate, in.EndDate, in.SortOrder))
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	return p, nil
}

func (r *ProjectRepository) Delete(ctx context.Context, resumeID, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1 AND resume_id = $2`, id, resumeID)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ProjectRepository) Reorder(ctx context.Context, resumeID string, orderedIDs []string) error {
	return reorder(ctx, r.pool, "projects", "resume_id", resumeID, orderedIDs)
}
