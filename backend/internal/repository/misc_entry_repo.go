package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MiscEntry struct {
	ID          string     `json:"id"`
	ResumeID    string     `json:"resume_id"`
	Kind        string     `json:"kind"` // award | publication | volunteer
	Title       string     `json:"title"`
	IssuerOrOrg *string    `json:"issuer_or_org"`
	URL         *string    `json:"url"`
	EntryDate   *time.Time `json:"entry_date"`
	Description *string    `json:"description"`
	SortOrder   int        `json:"sort_order"`
}

type MiscEntryInput struct {
	Kind        string     `json:"kind"`
	Title       string     `json:"title"`
	IssuerOrOrg *string    `json:"issuer_or_org"`
	URL         *string    `json:"url"`
	EntryDate   *time.Time `json:"entry_date"`
	Description *string    `json:"description"`
	SortOrder   int        `json:"sort_order"`
	ClientID    *string    `json:"client_id"`
}

type MiscEntryRepository struct {
	pool *pgxpool.Pool
}

func NewMiscEntryRepository(pool *pgxpool.Pool) *MiscEntryRepository {
	return &MiscEntryRepository{pool: pool}
}

const miscEntryColumns = `id, resume_id, kind, title, issuer_or_org, url, entry_date, description, sort_order`

func scanMiscEntry(row pgx.Row) (*MiscEntry, error) {
	var m MiscEntry
	err := row.Scan(&m.ID, &m.ResumeID, &m.Kind, &m.Title, &m.IssuerOrOrg, &m.URL, &m.EntryDate, &m.Description, &m.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MiscEntryRepository) List(ctx context.Context, resumeID string) ([]MiscEntry, error) {
	const q = `SELECT ` + miscEntryColumns + ` FROM misc_entries WHERE resume_id = $1 ORDER BY sort_order`
	rows, err := r.pool.Query(ctx, q, resumeID)
	if err != nil {
		return nil, fmt.Errorf("list misc entries: %w", err)
	}
	defer rows.Close()

	items := []MiscEntry{}
	for rows.Next() {
		m, err := scanMiscEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan misc entry: %w", err)
		}
		items = append(items, *m)
	}
	return items, rows.Err()
}

func (r *MiscEntryRepository) Create(ctx context.Context, resumeID string, in MiscEntryInput) (*MiscEntry, error) {
	q := `
		INSERT INTO misc_entries (resume_id, kind, title, issuer_or_org, url, entry_date, description, sort_order, client_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		` + onConflictClientID("resume_id") + `
		RETURNING ` + miscEntryColumns

	m, err := scanMiscEntry(r.pool.QueryRow(ctx, q, resumeID, in.Kind, in.Title, in.IssuerOrOrg, in.URL, in.EntryDate, in.Description, in.SortOrder, in.ClientID))
	if err != nil {
		return nil, fmt.Errorf("create misc entry: %w", err)
	}
	return m, nil
}

func (r *MiscEntryRepository) Update(ctx context.Context, resumeID, id string, in MiscEntryInput) (*MiscEntry, error) {
	const q = `
		UPDATE misc_entries SET kind = $3, title = $4, issuer_or_org = $5, url = $6, entry_date = $7, description = $8, sort_order = $9
		WHERE id = $1 AND resume_id = $2
		RETURNING ` + miscEntryColumns

	m, err := scanMiscEntry(r.pool.QueryRow(ctx, q, id, resumeID, in.Kind, in.Title, in.IssuerOrOrg, in.URL, in.EntryDate, in.Description, in.SortOrder))
	if err != nil {
		return nil, fmt.Errorf("update misc entry: %w", err)
	}
	return m, nil
}

func (r *MiscEntryRepository) Delete(ctx context.Context, resumeID, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM misc_entries WHERE id = $1 AND resume_id = $2`, id, resumeID)
	if err != nil {
		return fmt.Errorf("delete misc entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MiscEntryRepository) Reorder(ctx context.Context, resumeID string, orderedIDs []string) error {
	return reorder(ctx, r.pool, "misc_entries", "resume_id", resumeID, orderedIDs)
}
