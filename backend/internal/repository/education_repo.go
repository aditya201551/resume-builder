package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Education struct {
	ID                string     `json:"id"`
	ResumeID          string     `json:"resume_id"`
	Institution       string     `json:"institution"`
	Degree            *string    `json:"degree"`
	FieldOfStudy      *string    `json:"field_of_study"`
	Location          *string    `json:"location"`
	StartDate         *time.Time `json:"start_date"`
	EndDate           *time.Time `json:"end_date"`
	GPA               *string    `json:"gpa"`
	HonorsDescription *string    `json:"honors_description"`
	SortOrder         int        `json:"sort_order"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type EducationInput struct {
	Institution       string     `json:"institution"`
	Degree            *string    `json:"degree"`
	FieldOfStudy      *string    `json:"field_of_study"`
	Location          *string    `json:"location"`
	StartDate         *time.Time `json:"start_date"`
	EndDate           *time.Time `json:"end_date"`
	GPA               *string    `json:"gpa"`
	HonorsDescription *string    `json:"honors_description"`
	SortOrder         int        `json:"sort_order"`
	ClientID          *string    `json:"client_id"`
}

type EducationRepository struct {
	pool *pgxpool.Pool
}

func NewEducationRepository(pool *pgxpool.Pool) *EducationRepository {
	return &EducationRepository{pool: pool}
}

const educationColumns = `id, resume_id, institution, degree, field_of_study, location, start_date, end_date, gpa, honors_description, sort_order, created_at, updated_at`

func scanEducation(row pgx.Row) (*Education, error) {
	var e Education
	err := row.Scan(&e.ID, &e.ResumeID, &e.Institution, &e.Degree, &e.FieldOfStudy, &e.Location, &e.StartDate,
		&e.EndDate, &e.GPA, &e.HonorsDescription, &e.SortOrder, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EducationRepository) List(ctx context.Context, resumeID string) ([]Education, error) {
	const q = `SELECT ` + educationColumns + ` FROM educations WHERE resume_id = $1 ORDER BY sort_order`
	rows, err := r.pool.Query(ctx, q, resumeID)
	if err != nil {
		return nil, fmt.Errorf("list educations: %w", err)
	}
	defer rows.Close()

	items := []Education{}
	for rows.Next() {
		e, err := scanEducation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan education: %w", err)
		}
		items = append(items, *e)
	}
	return items, rows.Err()
}

func (r *EducationRepository) Create(ctx context.Context, resumeID string, in EducationInput) (*Education, error) {
	q := `
		INSERT INTO educations (resume_id, institution, degree, field_of_study, location, start_date, end_date, gpa, honors_description, sort_order, client_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		` + onConflictClientID("resume_id") + `
		RETURNING ` + educationColumns

	e, err := scanEducation(r.pool.QueryRow(ctx, q, resumeID, in.Institution, in.Degree, in.FieldOfStudy, in.Location,
		in.StartDate, in.EndDate, in.GPA, in.HonorsDescription, in.SortOrder, in.ClientID))
	if err != nil {
		return nil, fmt.Errorf("create education: %w", err)
	}
	return e, nil
}

func (r *EducationRepository) Update(ctx context.Context, resumeID, id string, in EducationInput) (*Education, error) {
	const q = `
		UPDATE educations SET
			institution = $3, degree = $4, field_of_study = $5, location = $6, start_date = $7, end_date = $8,
			gpa = $9, honors_description = $10, sort_order = $11, updated_at = now()
		WHERE id = $1 AND resume_id = $2
		RETURNING ` + educationColumns

	e, err := scanEducation(r.pool.QueryRow(ctx, q, id, resumeID, in.Institution, in.Degree, in.FieldOfStudy,
		in.Location, in.StartDate, in.EndDate, in.GPA, in.HonorsDescription, in.SortOrder))
	if err != nil {
		return nil, fmt.Errorf("update education: %w", err)
	}
	return e, nil
}

func (r *EducationRepository) Delete(ctx context.Context, resumeID, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM educations WHERE id = $1 AND resume_id = $2`, id, resumeID)
	if err != nil {
		return fmt.Errorf("delete education: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *EducationRepository) Reorder(ctx context.Context, resumeID string, orderedIDs []string) error {
	return reorder(ctx, r.pool, "educations", "resume_id", resumeID, orderedIDs)
}
