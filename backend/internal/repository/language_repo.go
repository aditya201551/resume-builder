package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Language struct {
	ID          string  `json:"id"`
	ResumeID    string  `json:"resume_id"`
	Name        string  `json:"name"`
	Proficiency *string `json:"proficiency"`
	SortOrder   int     `json:"sort_order"`
}

type LanguageInput struct {
	Name        string  `json:"name"`
	Proficiency *string `json:"proficiency"`
	SortOrder   int     `json:"sort_order"`
	ClientID    *string `json:"client_id"`
}

type LanguageRepository struct {
	pool *pgxpool.Pool
}

func NewLanguageRepository(pool *pgxpool.Pool) *LanguageRepository {
	return &LanguageRepository{pool: pool}
}

const languageColumns = `id, resume_id, name, proficiency, sort_order`

func scanLanguage(row pgx.Row) (*Language, error) {
	var l Language
	err := row.Scan(&l.ID, &l.ResumeID, &l.Name, &l.Proficiency, &l.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *LanguageRepository) List(ctx context.Context, resumeID string) ([]Language, error) {
	const q = `SELECT ` + languageColumns + ` FROM languages WHERE resume_id = $1 ORDER BY sort_order`
	rows, err := r.pool.Query(ctx, q, resumeID)
	if err != nil {
		return nil, fmt.Errorf("list languages: %w", err)
	}
	defer rows.Close()

	items := []Language{}
	for rows.Next() {
		l, err := scanLanguage(rows)
		if err != nil {
			return nil, fmt.Errorf("scan language: %w", err)
		}
		items = append(items, *l)
	}
	return items, rows.Err()
}

func (r *LanguageRepository) Create(ctx context.Context, resumeID string, in LanguageInput) (*Language, error) {
	q := `
		INSERT INTO languages (resume_id, name, proficiency, sort_order, client_id)
		VALUES ($1, $2, $3, $4, $5)
		` + onConflictClientID("resume_id") + `
		RETURNING ` + languageColumns

	l, err := scanLanguage(r.pool.QueryRow(ctx, q, resumeID, in.Name, in.Proficiency, in.SortOrder, in.ClientID))
	if err != nil {
		return nil, fmt.Errorf("create language: %w", err)
	}
	return l, nil
}

func (r *LanguageRepository) Update(ctx context.Context, resumeID, id string, in LanguageInput) (*Language, error) {
	const q = `
		UPDATE languages SET name = $3, proficiency = $4, sort_order = $5
		WHERE id = $1 AND resume_id = $2
		RETURNING ` + languageColumns

	l, err := scanLanguage(r.pool.QueryRow(ctx, q, id, resumeID, in.Name, in.Proficiency, in.SortOrder))
	if err != nil {
		return nil, fmt.Errorf("update language: %w", err)
	}
	return l, nil
}

func (r *LanguageRepository) Delete(ctx context.Context, resumeID, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM languages WHERE id = $1 AND resume_id = $2`, id, resumeID)
	if err != nil {
		return fmt.Errorf("delete language: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *LanguageRepository) Reorder(ctx context.Context, resumeID string, orderedIDs []string) error {
	return reorder(ctx, r.pool, "languages", "resume_id", resumeID, orderedIDs)
}
