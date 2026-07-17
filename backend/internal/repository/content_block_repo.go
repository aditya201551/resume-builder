package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ContentBlock is a read-side view spanning every rich-text Markdown field in
// the schema (currently work_experiences.content and projects.content) — the
// one granularity Phase 2's AI rewrite and Phase 3's keyword-gap matching
// both operate on, regardless of which table the text actually lives in.
type ContentBlock struct {
	Kind      string `json:"kind"` // "work_experience" | "project"
	ID        string `json:"id"`
	Label     string `json:"label"`
	Content   string `json:"content"`
	SortOrder int    `json:"sort_order"`
}

type ContentBlockRepository struct {
	pool            *pgxpool.Pool
	workExperiences *WorkExperienceRepository
	projects        *ProjectRepository
}

func NewContentBlockRepository(pool *pgxpool.Pool, workExperiences *WorkExperienceRepository, projects *ProjectRepository) *ContentBlockRepository {
	return &ContentBlockRepository{pool: pool, workExperiences: workExperiences, projects: projects}
}

func (r *ContentBlockRepository) List(ctx context.Context, resumeID string) ([]ContentBlock, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT 'work_experience', id, company || ' — ' || title, content, sort_order FROM work_experiences WHERE resume_id = $1
		 UNION ALL
		 SELECT 'project', id, name, content, sort_order FROM projects WHERE resume_id = $1`,
		resumeID)
	if err != nil {
		return nil, fmt.Errorf("list content blocks: %w", err)
	}
	defer rows.Close()

	blocks := []ContentBlock{}
	for rows.Next() {
		var b ContentBlock
		if err := rows.Scan(&b.Kind, &b.ID, &b.Label, &b.Content, &b.SortOrder); err != nil {
			return nil, fmt.Errorf("scan content block: %w", err)
		}
		blocks = append(blocks, b)
	}
	return blocks, rows.Err()
}

// UpdateContent writes the markdown body back to whichever table kind
// identifies. kind is validated by the service layer before this is called.
func (r *ContentBlockRepository) UpdateContent(ctx context.Context, resumeID, kind, blockID, content string) error {
	switch kind {
	case "work_experience":
		return r.workExperiences.UpdateContent(ctx, resumeID, blockID, content)
	case "project":
		return r.projects.UpdateContent(ctx, resumeID, blockID, content)
	default:
		return fmt.Errorf("unknown content block kind: %s", kind)
	}
}
