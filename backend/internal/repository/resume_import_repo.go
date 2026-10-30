package repository

import (
	"context"
	"encoding/json"

	"resume-builder/backend/internal/design"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ImportedResume contains content only: ownership, IDs and design are assigned by
// the server. No model-provided ID is used to select or change an existing row.
type ImportedResume struct {
	Resume          ResumeMetaInput         `json:"resume"`
	WorkExperiences []WorkExperienceInput   `json:"work_experiences"`
	Educations      []EducationInput        `json:"educations"`
	SkillGroups     []ImportedSkillGroup    `json:"skill_groups"`
	Projects        []ProjectInput          `json:"projects"`
	Certifications  []CertificationInput    `json:"certifications"`
	Languages       []LanguageInput         `json:"languages"`
	MiscEntries     []MiscEntryInput        `json:"misc_entries"`
	CustomSections  []ImportedCustomSection `json:"custom_sections"`
}

type ImportedSkillGroup struct {
	Name  string           `json:"name"`
	Items []SkillItemInput `json:"items"`
}

type ImportedCustomSection struct {
	Title   string                    `json:"title"`
	Entries []CustomSectionEntryInput `json:"entries"`
}

type ResumeImportRepository struct{ pool dbExecutor }

func NewResumeImportRepository(pool *pgxpool.Pool) *ResumeImportRepository {
	return &ResumeImportRepository{pool: pool}
}

func (r *ResumeImportRepository) Create(ctx context.Context, userID string, in ImportedResume) (*Resume, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.WithoutCancel(ctx))
	resume, err := (&ResumeRepository{pool: tx}).Create(ctx, userID, in.Resume)
	if err != nil {
		return nil, err
	}
	id := resume.ID
	for i, entry := range in.WorkExperiences {
		entry.SortOrder, entry.ClientID = i, nil
		if _, err = (&WorkExperienceRepository{pool: tx}).Create(ctx, id, entry); err != nil {
			return nil, err
		}
	}
	for i, entry := range in.Educations {
		entry.SortOrder, entry.ClientID = i, nil
		if _, err = (&EducationRepository{pool: tx}).Create(ctx, id, entry); err != nil {
			return nil, err
		}
	}
	for i, entry := range in.Projects {
		entry.SortOrder, entry.ClientID = i, nil
		if _, err = (&ProjectRepository{pool: tx}).Create(ctx, id, entry); err != nil {
			return nil, err
		}
	}
	for i, entry := range in.Certifications {
		entry.SortOrder, entry.ClientID = i, nil
		if _, err = (&CertificationRepository{pool: tx}).Create(ctx, id, entry); err != nil {
			return nil, err
		}
	}
	for i, entry := range in.Languages {
		entry.SortOrder, entry.ClientID = i, nil
		if _, err = (&LanguageRepository{pool: tx}).Create(ctx, id, entry); err != nil {
			return nil, err
		}
	}
	for i, entry := range in.MiscEntries {
		entry.SortOrder, entry.ClientID = i, nil
		if _, err = (&MiscEntryRepository{pool: tx}).Create(ctx, id, entry); err != nil {
			return nil, err
		}
	}
	skills := &SkillRepository{pool: tx}
	for i, group := range in.SkillGroups {
		created, e := skills.CreateGroup(ctx, id, SkillGroupInput{GroupName: group.Name, SortOrder: i})
		if e != nil {
			return nil, e
		}
		for j, entry := range group.Items {
			entry.SortOrder, entry.ClientID = j, nil
			if _, err = skills.CreateItem(ctx, id, created.ID, entry); err != nil {
				return nil, err
			}
		}
	}
	d := design.Default(design.DefaultTemplateID())
	custom := &CustomSectionRepository{pool: tx}
	for i, section := range in.CustomSections {
		created, e := custom.CreateSection(ctx, id, CustomSectionInput{Title: section.Title, SortOrder: i})
		if e != nil {
			return nil, e
		}
		for j, entry := range section.Entries {
			entry.SortOrder, entry.ClientID = j, nil
			if _, err = custom.CreateEntry(ctx, id, created.ID, entry); err != nil {
				return nil, err
			}
		}
	}
	raw, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	if _, err = (&ResumeDesignRepository{pool: tx}).Upsert(ctx, id, raw); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return resume, nil
}
