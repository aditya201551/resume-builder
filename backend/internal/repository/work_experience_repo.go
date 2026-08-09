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

type WorkExperience struct {
	ID             string          `json:"id"`
	ResumeID       string          `json:"resume_id"`
	Company        string          `json:"company"`
	CompanyURL     *string         `json:"company_url"`
	Title          string          `json:"title"`
	Location       *string         `json:"location"`
	EmploymentType *string         `json:"employment_type"`
	StartDate      *time.Time      `json:"start_date"`
	EndDate        *time.Time      `json:"end_date"`
	IsCurrent      bool            `json:"is_current"`
	Content        string          `json:"content"`
	Technologies   json.RawMessage `json:"technologies"`
	SortOrder      int             `json:"sort_order"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type WorkExperienceInput struct {
	Company        string          `json:"company"`
	CompanyURL     *string         `json:"company_url"`
	Title          string          `json:"title"`
	Location       *string         `json:"location"`
	EmploymentType *string         `json:"employment_type"`
	StartDate      *time.Time      `json:"start_date"`
	EndDate        *time.Time      `json:"end_date"`
	IsCurrent      bool            `json:"is_current"`
	Content        string          `json:"content"`
	Technologies   json.RawMessage `json:"technologies"`
	SortOrder      int             `json:"sort_order"`
	ClientID       *string         `json:"client_id"`
}

type WorkExperienceRepository struct {
	pool *pgxpool.Pool
}

func NewWorkExperienceRepository(pool *pgxpool.Pool) *WorkExperienceRepository {
	return &WorkExperienceRepository{pool: pool}
}

const workExperienceColumns = `id, resume_id, company, company_url, title, location, employment_type, start_date, end_date, is_current, content, technologies, sort_order, created_at, updated_at`

func scanWorkExperience(row pgx.Row) (*WorkExperience, error) {
	var w WorkExperience
	err := row.Scan(&w.ID, &w.ResumeID, &w.Company, &w.CompanyURL, &w.Title, &w.Location, &w.EmploymentType, &w.StartDate,
		&w.EndDate, &w.IsCurrent, &w.Content, &w.Technologies, &w.SortOrder, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *WorkExperienceRepository) List(ctx context.Context, resumeID string) ([]WorkExperience, error) {
	const q = `SELECT ` + workExperienceColumns + ` FROM work_experiences WHERE resume_id = $1 ORDER BY sort_order`
	rows, err := r.pool.Query(ctx, q, resumeID)
	if err != nil {
		return nil, fmt.Errorf("list work experiences: %w", err)
	}
	defer rows.Close()

	items := []WorkExperience{}
	for rows.Next() {
		w, err := scanWorkExperience(rows)
		if err != nil {
			return nil, fmt.Errorf("scan work experience: %w", err)
		}
		items = append(items, *w)
	}
	return items, rows.Err()
}

func (r *WorkExperienceRepository) Create(ctx context.Context, resumeID string, in WorkExperienceInput) (*WorkExperience, error) {
	q := `
		INSERT INTO work_experiences (resume_id, company, company_url, title, location, employment_type, start_date, end_date, is_current, content, technologies, sort_order, client_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		` + onConflictClientID("resume_id") + `
		RETURNING ` + workExperienceColumns

	w, err := scanWorkExperience(r.pool.QueryRow(ctx, q, resumeID, in.Company, in.CompanyURL, in.Title, in.Location, in.EmploymentType,
		in.StartDate, in.EndDate, in.IsCurrent, in.Content, defaultJSON(in.Technologies, `[]`), in.SortOrder, in.ClientID))
	if err != nil {
		return nil, fmt.Errorf("create work experience: %w", err)
	}
	return w, nil
}

func (r *WorkExperienceRepository) Update(ctx context.Context, resumeID, id string, in WorkExperienceInput) (*WorkExperience, error) {
	const q = `
		UPDATE work_experiences SET
			company = $3, company_url = $4, title = $5, location = $6, employment_type = $7, start_date = $8, end_date = $9,
			is_current = $10, content = $11, technologies = $12, sort_order = $13, updated_at = now()
		WHERE id = $1 AND resume_id = $2
		RETURNING ` + workExperienceColumns

	w, err := scanWorkExperience(r.pool.QueryRow(ctx, q, id, resumeID, in.Company, in.CompanyURL, in.Title, in.Location,
		in.EmploymentType, in.StartDate, in.EndDate, in.IsCurrent, in.Content, defaultJSON(in.Technologies, `[]`), in.SortOrder))
	if err != nil {
		return nil, fmt.Errorf("update work experience: %w", err)
	}
	return w, nil
}

func (r *WorkExperienceRepository) Delete(ctx context.Context, resumeID, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM work_experiences WHERE id = $1 AND resume_id = $2`, id, resumeID)
	if err != nil {
		return fmt.Errorf("delete work experience: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *WorkExperienceRepository) Reorder(ctx context.Context, resumeID string, orderedIDs []string) error {
	return reorder(ctx, r.pool, "work_experiences", "resume_id", resumeID, orderedIDs)
}
