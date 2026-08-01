package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CustomSectionEntry struct {
	ID          string     `json:"id"`
	SectionID   string     `json:"custom_section_id"`
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	EntryDate   *time.Time `json:"entry_date"`
	SortOrder   int        `json:"sort_order"`
}

type CustomSectionEntryInput struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	EntryDate   *time.Time `json:"entry_date"`
	SortOrder   int        `json:"sort_order"`
	ClientID    *string    `json:"client_id"`
}

type CustomSection struct {
	ID        string               `json:"id"`
	ResumeID  string               `json:"resume_id"`
	Title     string               `json:"title"`
	SortOrder int                  `json:"sort_order"`
	Entries   []CustomSectionEntry `json:"entries"`
}

type CustomSectionInput struct {
	Title     string  `json:"title"`
	SortOrder int     `json:"sort_order"`
	ClientID  *string `json:"client_id"`
}

type CustomSectionRepository struct {
	pool *pgxpool.Pool
}

func NewCustomSectionRepository(pool *pgxpool.Pool) *CustomSectionRepository {
	return &CustomSectionRepository{pool: pool}
}

func (r *CustomSectionRepository) ListSections(ctx context.Context, resumeID string) ([]CustomSection, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, resume_id, title, sort_order FROM custom_sections WHERE resume_id = $1 ORDER BY sort_order`, resumeID)
	if err != nil {
		return nil, fmt.Errorf("list custom sections: %w", err)
	}
	sections := []CustomSection{}
	for rows.Next() {
		var s CustomSection
		if err := rows.Scan(&s.ID, &s.ResumeID, &s.Title, &s.SortOrder); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan custom section: %w", err)
		}
		s.Entries = []CustomSectionEntry{}
		sections = append(sections, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range sections {
		entries, err := r.listEntries(ctx, sections[i].ID)
		if err != nil {
			return nil, err
		}
		sections[i].Entries = entries
	}
	return sections, nil
}

func (r *CustomSectionRepository) listEntries(ctx context.Context, sectionID string) ([]CustomSectionEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, custom_section_id, title, description, entry_date, sort_order FROM custom_section_entries WHERE custom_section_id = $1 ORDER BY sort_order`,
		sectionID)
	if err != nil {
		return nil, fmt.Errorf("list custom section entries: %w", err)
	}
	defer rows.Close()

	entries := []CustomSectionEntry{}
	for rows.Next() {
		var e CustomSectionEntry
		if err := rows.Scan(&e.ID, &e.SectionID, &e.Title, &e.Description, &e.EntryDate, &e.SortOrder); err != nil {
			return nil, fmt.Errorf("scan custom section entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *CustomSectionRepository) CreateSection(ctx context.Context, resumeID string, in CustomSectionInput) (*CustomSection, error) {
	q := `INSERT INTO custom_sections (resume_id, title, sort_order, client_id) VALUES ($1, $2, $3, $4)
		` + onConflictClientID("resume_id") + `
		RETURNING id, resume_id, title, sort_order`
	var s CustomSection
	err := r.pool.QueryRow(ctx, q, resumeID, in.Title, in.SortOrder, in.ClientID).Scan(&s.ID, &s.ResumeID, &s.Title, &s.SortOrder)
	if err != nil {
		return nil, fmt.Errorf("create custom section: %w", err)
	}
	s.Entries = []CustomSectionEntry{}
	return &s, nil
}

func (r *CustomSectionRepository) UpdateSection(ctx context.Context, resumeID, sectionID string, in CustomSectionInput) (*CustomSection, error) {
	var s CustomSection
	err := r.pool.QueryRow(ctx,
		`UPDATE custom_sections SET title = $3, sort_order = $4 WHERE id = $1 AND resume_id = $2 RETURNING id, resume_id, title, sort_order`,
		sectionID, resumeID, in.Title, in.SortOrder).Scan(&s.ID, &s.ResumeID, &s.Title, &s.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update custom section: %w", err)
	}
	entries, err := r.listEntries(ctx, sectionID)
	if err != nil {
		return nil, err
	}
	s.Entries = entries
	return &s, nil
}

func (r *CustomSectionRepository) DeleteSection(ctx context.Context, resumeID, sectionID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM custom_sections WHERE id = $1 AND resume_id = $2`, sectionID, resumeID)
	if err != nil {
		return fmt.Errorf("delete custom section: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CustomSectionRepository) ReorderSections(ctx context.Context, resumeID string, orderedIDs []string) error {
	return reorder(ctx, r.pool, "custom_sections", "resume_id", resumeID, orderedIDs)
}

func (r *CustomSectionRepository) sectionBelongsToResume(ctx context.Context, resumeID, sectionID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM custom_sections WHERE id = $1 AND resume_id = $2)`, sectionID, resumeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check custom section ownership: %w", err)
	}
	return exists, nil
}

func (r *CustomSectionRepository) CreateEntry(ctx context.Context, resumeID, sectionID string, in CustomSectionEntryInput) (*CustomSectionEntry, error) {
	ok, err := r.sectionBelongsToResume(ctx, resumeID, sectionID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}

	q := `INSERT INTO custom_section_entries (custom_section_id, title, description, entry_date, sort_order, client_id) VALUES ($1, $2, $3, $4, $5, $6)
		 ` + onConflictClientID("custom_section_id") + `
		 RETURNING id, custom_section_id, title, description, entry_date, sort_order`
	var e CustomSectionEntry
	err = r.pool.QueryRow(ctx, q, sectionID, in.Title, in.Description, in.EntryDate, in.SortOrder, in.ClientID).Scan(&e.ID, &e.SectionID, &e.Title, &e.Description, &e.EntryDate, &e.SortOrder)
	if err != nil {
		return nil, fmt.Errorf("create custom section entry: %w", err)
	}
	return &e, nil
}

func (r *CustomSectionRepository) UpdateEntry(ctx context.Context, resumeID, sectionID, entryID string, in CustomSectionEntryInput) (*CustomSectionEntry, error) {
	ok, err := r.sectionBelongsToResume(ctx, resumeID, sectionID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}

	var e CustomSectionEntry
	err = r.pool.QueryRow(ctx,
		`UPDATE custom_section_entries SET title = $3, description = $4, entry_date = $5, sort_order = $6
		 WHERE id = $1 AND custom_section_id = $2
		 RETURNING id, custom_section_id, title, description, entry_date, sort_order`,
		entryID, sectionID, in.Title, in.Description, in.EntryDate, in.SortOrder,
	).Scan(&e.ID, &e.SectionID, &e.Title, &e.Description, &e.EntryDate, &e.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update custom section entry: %w", err)
	}
	return &e, nil
}

func (r *CustomSectionRepository) DeleteEntry(ctx context.Context, resumeID, sectionID, entryID string) error {
	ok, err := r.sectionBelongsToResume(ctx, resumeID, sectionID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}

	tag, err := r.pool.Exec(ctx, `DELETE FROM custom_section_entries WHERE id = $1 AND custom_section_id = $2`, entryID, sectionID)
	if err != nil {
		return fmt.Errorf("delete custom section entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CustomSectionRepository) ReorderEntries(ctx context.Context, resumeID, sectionID string, orderedIDs []string) error {
	ok, err := r.sectionBelongsToResume(ctx, resumeID, sectionID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return reorder(ctx, r.pool, "custom_section_entries", "custom_section_id", sectionID, orderedIDs)
}
