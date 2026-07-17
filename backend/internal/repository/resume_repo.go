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

type Resume struct {
	ID             string          `json:"id"`
	UserID         string          `json:"user_id"`
	Label          string          `json:"label"`
	TemplateID     *string         `json:"template_id"`
	FullName       string          `json:"full_name"`
	Headline       *string         `json:"headline"`
	Email          *string         `json:"email"`
	Phone          *string         `json:"phone"`
	Location       *string         `json:"location"`
	PhotoURL       *string         `json:"photo_url"`
	Summary        *string         `json:"summary"`
	Links          json.RawMessage `json:"links"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	LastExportedAt *time.Time      `json:"last_exported_at"`
}

// ResumeMetaInput is the subset of Resume fields a client may set directly;
// UserID/timestamps/LastExportedAt are server-controlled.
type ResumeMetaInput struct {
	Label      string          `json:"label"`
	TemplateID *string         `json:"template_id"`
	FullName   string          `json:"full_name"`
	Headline   *string         `json:"headline"`
	Email      *string         `json:"email"`
	Phone      *string         `json:"phone"`
	Location   *string         `json:"location"`
	PhotoURL   *string         `json:"photo_url"`
	Summary    *string         `json:"summary"`
	Links      json.RawMessage `json:"links"`
}

type ResumeRepository struct {
	pool *pgxpool.Pool
}

func NewResumeRepository(pool *pgxpool.Pool) *ResumeRepository {
	return &ResumeRepository{pool: pool}
}

const resumeColumns = `id, user_id, label, template_id, full_name, headline, email, phone, location, photo_url, summary, links, created_at, updated_at, last_exported_at`

func scanResume(row pgx.Row) (*Resume, error) {
	var r Resume
	err := row.Scan(&r.ID, &r.UserID, &r.Label, &r.TemplateID, &r.FullName, &r.Headline, &r.Email, &r.Phone,
		&r.Location, &r.PhotoURL, &r.Summary, &r.Links, &r.CreatedAt, &r.UpdatedAt, &r.LastExportedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// Create inserts the resume and, when a template is given, seeds
// resume_section_configs from that template's default_region_map so the
// resume renders correctly without a separate "pick layout" step.
func (r *ResumeRepository) Create(ctx context.Context, userID string, in ResumeMetaInput) (*Resume, error) {
	links := defaultJSON(in.Links, `[]`)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const insertResume = `
		INSERT INTO resumes (user_id, label, template_id, full_name, headline, email, phone, location, photo_url, summary, links)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING ` + resumeColumns

	resume, err := scanResume(tx.QueryRow(ctx, insertResume, userID, in.Label, in.TemplateID, in.FullName, in.Headline,
		in.Email, in.Phone, in.Location, in.PhotoURL, in.Summary, links))
	if err != nil {
		return nil, fmt.Errorf("create resume: %w", err)
	}

	if in.TemplateID != nil {
		var regionMap json.RawMessage
		if err := tx.QueryRow(ctx, `SELECT default_region_map FROM templates WHERE id = $1`, *in.TemplateID).Scan(&regionMap); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, fmt.Errorf("find template: %w", err)
		}

		var regions map[string]string
		if err := json.Unmarshal(regionMap, &regions); err != nil {
			return nil, fmt.Errorf("parse template regions: %w", err)
		}

		const insertConfig = `
			INSERT INTO resume_section_configs (resume_id, section_type, region, is_visible, sort_order)
			VALUES ($1, $2, $3, true, $4)`
		i := 0
		for sectionType, region := range regions {
			if _, err := tx.Exec(ctx, insertConfig, resume.ID, sectionType, region, i); err != nil {
				return nil, fmt.Errorf("seed section config for %s: %w", sectionType, err)
			}
			i++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return resume, nil
}

func (r *ResumeRepository) FindByID(ctx context.Context, id string) (*Resume, error) {
	const q = `SELECT ` + resumeColumns + ` FROM resumes WHERE id = $1`
	resume, err := scanResume(r.pool.QueryRow(ctx, q, id))
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("find resume: %w", err)
	}
	return resume, err
}

func (r *ResumeRepository) ListByUser(ctx context.Context, userID string) ([]Resume, error) {
	const q = `SELECT ` + resumeColumns + ` FROM resumes WHERE user_id = $1 ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list resumes: %w", err)
	}
	defer rows.Close()

	resumes := []Resume{}
	for rows.Next() {
		resume, err := scanResume(rows)
		if err != nil {
			return nil, fmt.Errorf("scan resume: %w", err)
		}
		resumes = append(resumes, *resume)
	}
	return resumes, rows.Err()
}

func (r *ResumeRepository) UpdateMeta(ctx context.Context, id string, in ResumeMetaInput) (*Resume, error) {
	const q = `
		UPDATE resumes SET
			label = $2, template_id = $3, full_name = $4, headline = $5, email = $6,
			phone = $7, location = $8, photo_url = $9, summary = $10, links = $11, updated_at = now()
		WHERE id = $1
		RETURNING ` + resumeColumns

	resume, err := scanResume(r.pool.QueryRow(ctx, q, id, in.Label, in.TemplateID, in.FullName, in.Headline,
		in.Email, in.Phone, in.Location, in.PhotoURL, in.Summary, defaultJSON(in.Links, `[]`)))
	if err != nil {
		return nil, fmt.Errorf("update resume: %w", err)
	}
	return resume, nil
}

func (r *ResumeRepository) Delete(ctx context.Context, id string) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM resumes WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete resume: %w", err)
	}
	return nil
}

// SwitchTemplate points the resume at a new template and re-seeds every
// existing resume_section_configs row's region from that template's
// default_region_map, keyed by section_type — visibility, ordering, and
// title overrides are left untouched.
func (r *ResumeRepository) SwitchTemplate(ctx context.Context, resumeID, templateID string) (*Resume, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var regionMap json.RawMessage
	if err := tx.QueryRow(ctx, `SELECT default_region_map FROM templates WHERE id = $1`, templateID).Scan(&regionMap); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find template: %w", err)
	}

	var regions map[string]string
	if err := json.Unmarshal(regionMap, &regions); err != nil {
		return nil, fmt.Errorf("parse template regions: %w", err)
	}

	const updateResume = `UPDATE resumes SET template_id = $2, updated_at = now() WHERE id = $1 RETURNING ` + resumeColumns
	resume, err := scanResume(tx.QueryRow(ctx, updateResume, resumeID, templateID))
	if err != nil {
		return nil, fmt.Errorf("update resume template: %w", err)
	}

	const updateRegion = `UPDATE resume_section_configs SET region = $3 WHERE resume_id = $1 AND section_type = $2`
	for sectionType, region := range regions {
		if _, err := tx.Exec(ctx, updateRegion, resumeID, sectionType, region); err != nil {
			return nil, fmt.Errorf("reseed region for %s: %w", sectionType, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return resume, nil
}

// Duplicate deep-copies a resume and every child row into a brand-new
// resume owned by the same user, so editing one never affects the other.
func (r *ResumeRepository) Duplicate(ctx context.Context, resumeID string) (*Resume, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const cloneResume = `
		INSERT INTO resumes (user_id, label, template_id, full_name, headline, email, phone, location, photo_url, summary, links)
		SELECT user_id, label || ' (copy)', template_id, full_name, headline, email, phone, location, photo_url, summary, links
		FROM resumes WHERE id = $1
		RETURNING ` + resumeColumns

	newResume, err := scanResume(tx.QueryRow(ctx, cloneResume, resumeID))
	if err != nil {
		return nil, fmt.Errorf("clone resume: %w", err)
	}
	newID := newResume.ID

	simpleCloneStatements := []string{
		`INSERT INTO work_experiences (resume_id, company, title, location, employment_type, start_date, end_date, is_current, content, technologies, sort_order)
		 SELECT $2, company, title, location, employment_type, start_date, end_date, is_current, content, technologies, sort_order
		 FROM work_experiences WHERE resume_id = $1`,
		`INSERT INTO educations (resume_id, institution, degree, field_of_study, location, start_date, end_date, gpa, honors_description, sort_order)
		 SELECT $2, institution, degree, field_of_study, location, start_date, end_date, gpa, honors_description, sort_order
		 FROM educations WHERE resume_id = $1`,
		`INSERT INTO projects (resume_id, name, content, role, technologies, url, start_date, end_date, sort_order)
		 SELECT $2, name, content, role, technologies, url, start_date, end_date, sort_order
		 FROM projects WHERE resume_id = $1`,
		`INSERT INTO certifications (resume_id, name, issuer, issue_date, expiry_date, credential_url, sort_order)
		 SELECT $2, name, issuer, issue_date, expiry_date, credential_url, sort_order
		 FROM certifications WHERE resume_id = $1`,
		`INSERT INTO languages (resume_id, name, proficiency, sort_order)
		 SELECT $2, name, proficiency, sort_order
		 FROM languages WHERE resume_id = $1`,
		`INSERT INTO misc_entries (resume_id, kind, title, issuer_or_org, url, entry_date, description, sort_order)
		 SELECT $2, kind, title, issuer_or_org, url, entry_date, description, sort_order
		 FROM misc_entries WHERE resume_id = $1`,
	}
	for _, stmt := range simpleCloneStatements {
		if _, err := tx.Exec(ctx, stmt, resumeID, newID); err != nil {
			return nil, fmt.Errorf("clone child rows: %w", err)
		}
	}

	// skill_groups -> skill_items need an old-group-id -> new-group-id map.
	groupRows, err := tx.Query(ctx, `SELECT id, group_name, sort_order FROM skill_groups WHERE resume_id = $1`, resumeID)
	if err != nil {
		return nil, fmt.Errorf("list skill groups: %w", err)
	}
	type groupRow struct {
		id, name string
		sort     int
	}
	var groups []groupRow
	for groupRows.Next() {
		var g groupRow
		if err := groupRows.Scan(&g.id, &g.name, &g.sort); err != nil {
			groupRows.Close()
			return nil, fmt.Errorf("scan skill group: %w", err)
		}
		groups = append(groups, g)
	}
	groupRows.Close()
	if err := groupRows.Err(); err != nil {
		return nil, err
	}

	for _, g := range groups {
		var newGroupID string
		err := tx.QueryRow(ctx,
			`INSERT INTO skill_groups (resume_id, group_name, sort_order) VALUES ($1, $2, $3) RETURNING id`,
			newID, g.name, g.sort).Scan(&newGroupID)
		if err != nil {
			return nil, fmt.Errorf("clone skill group: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO skill_items (skill_group_id, name, proficiency, sort_order)
			 SELECT $2, name, proficiency, sort_order FROM skill_items WHERE skill_group_id = $1`,
			g.id, newGroupID); err != nil {
			return nil, fmt.Errorf("clone skill items: %w", err)
		}
	}

	// custom_sections -> custom_section_entries need the same remap, and
	// resume_section_configs.custom_section_id must follow it too.
	sectionRows, err := tx.Query(ctx, `SELECT id, title, sort_order FROM custom_sections WHERE resume_id = $1`, resumeID)
	if err != nil {
		return nil, fmt.Errorf("list custom sections: %w", err)
	}
	type sectionRow struct {
		id, title string
		sort      int
	}
	var sections []sectionRow
	for sectionRows.Next() {
		var s sectionRow
		if err := sectionRows.Scan(&s.id, &s.title, &s.sort); err != nil {
			sectionRows.Close()
			return nil, fmt.Errorf("scan custom section: %w", err)
		}
		sections = append(sections, s)
	}
	sectionRows.Close()
	if err := sectionRows.Err(); err != nil {
		return nil, err
	}

	customSectionIDMap := map[string]string{}
	for _, s := range sections {
		var newSectionID string
		err := tx.QueryRow(ctx,
			`INSERT INTO custom_sections (resume_id, title, sort_order) VALUES ($1, $2, $3) RETURNING id`,
			newID, s.title, s.sort).Scan(&newSectionID)
		if err != nil {
			return nil, fmt.Errorf("clone custom section: %w", err)
		}
		customSectionIDMap[s.id] = newSectionID

		if _, err := tx.Exec(ctx,
			`INSERT INTO custom_section_entries (custom_section_id, title, description, entry_date, sort_order)
			 SELECT $2, title, description, entry_date, sort_order FROM custom_section_entries WHERE custom_section_id = $1`,
			s.id, newSectionID); err != nil {
			return nil, fmt.Errorf("clone custom section entries: %w", err)
		}
	}

	configRows, err := tx.Query(ctx,
		`SELECT section_type, custom_section_id, region, is_visible, display_title_override, sort_order
		 FROM resume_section_configs WHERE resume_id = $1`, resumeID)
	if err != nil {
		return nil, fmt.Errorf("list section configs: %w", err)
	}
	type configRow struct {
		sectionType, region, titleOverride *string
		customSectionID                    *string
		isVisible                          bool
		sort                               int
	}
	var configs []configRow
	for configRows.Next() {
		var c configRow
		if err := configRows.Scan(&c.sectionType, &c.customSectionID, &c.region, &c.isVisible, &c.titleOverride, &c.sort); err != nil {
			configRows.Close()
			return nil, fmt.Errorf("scan section config: %w", err)
		}
		configs = append(configs, c)
	}
	configRows.Close()
	if err := configRows.Err(); err != nil {
		return nil, err
	}

	for _, c := range configs {
		var newCustomSectionID *string
		if c.customSectionID != nil {
			if mapped, ok := customSectionIDMap[*c.customSectionID]; ok {
				newCustomSectionID = &mapped
			}
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO resume_section_configs (resume_id, section_type, custom_section_id, region, is_visible, display_title_override, sort_order)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			newID, c.sectionType, newCustomSectionID, c.region, c.isVisible, c.titleOverride, c.sort); err != nil {
			return nil, fmt.Errorf("clone section config: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return newResume, nil
}
