package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SkillItem struct {
	ID          string  `json:"id"`
	GroupID     string  `json:"skill_group_id"`
	Name        string  `json:"name"`
	Proficiency *string `json:"proficiency"`
	SortOrder   int     `json:"sort_order"`
}

type SkillItemInput struct {
	Name        string  `json:"name"`
	Proficiency *string `json:"proficiency"`
	SortOrder   int     `json:"sort_order"`
}

type SkillGroup struct {
	ID        string      `json:"id"`
	ResumeID  string      `json:"resume_id"`
	GroupName string      `json:"group_name"`
	SortOrder int         `json:"sort_order"`
	Items     []SkillItem `json:"items"`
}

type SkillGroupInput struct {
	GroupName string `json:"group_name"`
	SortOrder int    `json:"sort_order"`
}

type SkillRepository struct {
	pool *pgxpool.Pool
}

func NewSkillRepository(pool *pgxpool.Pool) *SkillRepository {
	return &SkillRepository{pool: pool}
}

// ListGroups returns every skill group for the resume, each with its items
// loaded — the grouping is intrinsic to how skills render, so callers never
// want groups without their items.
func (r *SkillRepository) ListGroups(ctx context.Context, resumeID string) ([]SkillGroup, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, resume_id, group_name, sort_order FROM skill_groups WHERE resume_id = $1 ORDER BY sort_order`, resumeID)
	if err != nil {
		return nil, fmt.Errorf("list skill groups: %w", err)
	}
	groups := []SkillGroup{}
	for rows.Next() {
		var g SkillGroup
		if err := rows.Scan(&g.ID, &g.ResumeID, &g.GroupName, &g.SortOrder); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan skill group: %w", err)
		}
		g.Items = []SkillItem{}
		groups = append(groups, g)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range groups {
		items, err := r.listItems(ctx, groups[i].ID)
		if err != nil {
			return nil, err
		}
		groups[i].Items = items
	}
	return groups, nil
}

func (r *SkillRepository) listItems(ctx context.Context, groupID string) ([]SkillItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, skill_group_id, name, proficiency, sort_order FROM skill_items WHERE skill_group_id = $1 ORDER BY sort_order`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list skill items: %w", err)
	}
	defer rows.Close()

	items := []SkillItem{}
	for rows.Next() {
		var it SkillItem
		if err := rows.Scan(&it.ID, &it.GroupID, &it.Name, &it.Proficiency, &it.SortOrder); err != nil {
			return nil, fmt.Errorf("scan skill item: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (r *SkillRepository) CreateGroup(ctx context.Context, resumeID string, in SkillGroupInput) (*SkillGroup, error) {
	var g SkillGroup
	err := r.pool.QueryRow(ctx,
		`INSERT INTO skill_groups (resume_id, group_name, sort_order) VALUES ($1, $2, $3) RETURNING id, resume_id, group_name, sort_order`,
		resumeID, in.GroupName, in.SortOrder).Scan(&g.ID, &g.ResumeID, &g.GroupName, &g.SortOrder)
	if err != nil {
		return nil, fmt.Errorf("create skill group: %w", err)
	}
	g.Items = []SkillItem{}
	return &g, nil
}

func (r *SkillRepository) UpdateGroup(ctx context.Context, resumeID, groupID string, in SkillGroupInput) (*SkillGroup, error) {
	var g SkillGroup
	err := r.pool.QueryRow(ctx,
		`UPDATE skill_groups SET group_name = $3, sort_order = $4 WHERE id = $1 AND resume_id = $2 RETURNING id, resume_id, group_name, sort_order`,
		groupID, resumeID, in.GroupName, in.SortOrder).Scan(&g.ID, &g.ResumeID, &g.GroupName, &g.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update skill group: %w", err)
	}
	items, err := r.listItems(ctx, groupID)
	if err != nil {
		return nil, err
	}
	g.Items = items
	return &g, nil
}

func (r *SkillRepository) DeleteGroup(ctx context.Context, resumeID, groupID string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM skill_groups WHERE id = $1 AND resume_id = $2`, groupID, resumeID)
	if err != nil {
		return fmt.Errorf("delete skill group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SkillRepository) ReorderGroups(ctx context.Context, resumeID string, orderedIDs []string) error {
	return reorder(ctx, r.pool, "skill_groups", "resume_id", resumeID, orderedIDs)
}

// groupBelongsToResume guards item mutations: skill_items has no resume_id of
// its own, so ownership is proven by checking its parent group's resume_id.
func (r *SkillRepository) groupBelongsToResume(ctx context.Context, resumeID, groupID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM skill_groups WHERE id = $1 AND resume_id = $2)`, groupID, resumeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check skill group ownership: %w", err)
	}
	return exists, nil
}

func (r *SkillRepository) CreateItem(ctx context.Context, resumeID, groupID string, in SkillItemInput) (*SkillItem, error) {
	ok, err := r.groupBelongsToResume(ctx, resumeID, groupID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}

	var it SkillItem
	err = r.pool.QueryRow(ctx,
		`INSERT INTO skill_items (skill_group_id, name, proficiency, sort_order) VALUES ($1, $2, $3, $4) RETURNING id, skill_group_id, name, proficiency, sort_order`,
		groupID, in.Name, in.Proficiency, in.SortOrder).Scan(&it.ID, &it.GroupID, &it.Name, &it.Proficiency, &it.SortOrder)
	if err != nil {
		return nil, fmt.Errorf("create skill item: %w", err)
	}
	return &it, nil
}

func (r *SkillRepository) UpdateItem(ctx context.Context, resumeID, groupID, itemID string, in SkillItemInput) (*SkillItem, error) {
	ok, err := r.groupBelongsToResume(ctx, resumeID, groupID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}

	var it SkillItem
	err = r.pool.QueryRow(ctx,
		`UPDATE skill_items SET name = $3, proficiency = $4, sort_order = $5 WHERE id = $1 AND skill_group_id = $2 RETURNING id, skill_group_id, name, proficiency, sort_order`,
		itemID, groupID, in.Name, in.Proficiency, in.SortOrder).Scan(&it.ID, &it.GroupID, &it.Name, &it.Proficiency, &it.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update skill item: %w", err)
	}
	return &it, nil
}

func (r *SkillRepository) DeleteItem(ctx context.Context, resumeID, groupID, itemID string) error {
	ok, err := r.groupBelongsToResume(ctx, resumeID, groupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}

	tag, err := r.pool.Exec(ctx, `DELETE FROM skill_items WHERE id = $1 AND skill_group_id = $2`, itemID, groupID)
	if err != nil {
		return fmt.Errorf("delete skill item: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SkillRepository) ReorderItems(ctx context.Context, resumeID, groupID string, orderedIDs []string) error {
	ok, err := r.groupBelongsToResume(ctx, resumeID, groupID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return reorder(ctx, r.pool, "skill_items", "skill_group_id", groupID, orderedIDs)
}
