package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Certification struct {
	ID            string     `json:"id"`
	ResumeID      string     `json:"resume_id"`
	Name          string     `json:"name"`
	Issuer        *string    `json:"issuer"`
	IssueDate     *time.Time `json:"issue_date"`
	ExpiryDate    *time.Time `json:"expiry_date"`
	CredentialURL *string    `json:"credential_url"`
	SortOrder     int        `json:"sort_order"`
}

type CertificationInput struct {
	Name          string     `json:"name"`
	Issuer        *string    `json:"issuer"`
	IssueDate     *time.Time `json:"issue_date"`
	ExpiryDate    *time.Time `json:"expiry_date"`
	CredentialURL *string    `json:"credential_url"`
	SortOrder     int        `json:"sort_order"`
}

type CertificationRepository struct {
	pool *pgxpool.Pool
}

func NewCertificationRepository(pool *pgxpool.Pool) *CertificationRepository {
	return &CertificationRepository{pool: pool}
}

const certificationColumns = `id, resume_id, name, issuer, issue_date, expiry_date, credential_url, sort_order`

func scanCertification(row pgx.Row) (*Certification, error) {
	var c Certification
	err := row.Scan(&c.ID, &c.ResumeID, &c.Name, &c.Issuer, &c.IssueDate, &c.ExpiryDate, &c.CredentialURL, &c.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CertificationRepository) List(ctx context.Context, resumeID string) ([]Certification, error) {
	const q = `SELECT ` + certificationColumns + ` FROM certifications WHERE resume_id = $1 ORDER BY sort_order`
	rows, err := r.pool.Query(ctx, q, resumeID)
	if err != nil {
		return nil, fmt.Errorf("list certifications: %w", err)
	}
	defer rows.Close()

	items := []Certification{}
	for rows.Next() {
		c, err := scanCertification(rows)
		if err != nil {
			return nil, fmt.Errorf("scan certification: %w", err)
		}
		items = append(items, *c)
	}
	return items, rows.Err()
}

func (r *CertificationRepository) Create(ctx context.Context, resumeID string, in CertificationInput) (*Certification, error) {
	const q = `
		INSERT INTO certifications (resume_id, name, issuer, issue_date, expiry_date, credential_url, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + certificationColumns

	c, err := scanCertification(r.pool.QueryRow(ctx, q, resumeID, in.Name, in.Issuer, in.IssueDate, in.ExpiryDate, in.CredentialURL, in.SortOrder))
	if err != nil {
		return nil, fmt.Errorf("create certification: %w", err)
	}
	return c, nil
}

func (r *CertificationRepository) Update(ctx context.Context, resumeID, id string, in CertificationInput) (*Certification, error) {
	const q = `
		UPDATE certifications SET name = $3, issuer = $4, issue_date = $5, expiry_date = $6, credential_url = $7, sort_order = $8
		WHERE id = $1 AND resume_id = $2
		RETURNING ` + certificationColumns

	c, err := scanCertification(r.pool.QueryRow(ctx, q, id, resumeID, in.Name, in.Issuer, in.IssueDate, in.ExpiryDate, in.CredentialURL, in.SortOrder))
	if err != nil {
		return nil, fmt.Errorf("update certification: %w", err)
	}
	return c, nil
}

func (r *CertificationRepository) Delete(ctx context.Context, resumeID, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM certifications WHERE id = $1 AND resume_id = $2`, id, resumeID)
	if err != nil {
		return fmt.Errorf("delete certification: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *CertificationRepository) Reorder(ctx context.Context, resumeID string, orderedIDs []string) error {
	return reorder(ctx, r.pool, "certifications", "resume_id", resumeID, orderedIDs)
}
