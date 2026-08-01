package service

import (
	"context"

	"resume-builder/backend/internal/repository"
)

type CustomSectionService struct {
	resumes *repository.ResumeRepository
	configs *repository.SectionConfigRepository
	repo    *repository.CustomSectionRepository
}

func NewCustomSectionService(resumes *repository.ResumeRepository, configs *repository.SectionConfigRepository, repo *repository.CustomSectionRepository) *CustomSectionService {
	return &CustomSectionService{resumes: resumes, configs: configs, repo: repo}
}

func (s *CustomSectionService) ListSections(ctx context.Context, callerUserID, resumeID string) ([]repository.CustomSection, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	return s.repo.ListSections(ctx, resumeID)
}

// CreateSection also seeds a matching resume_section_configs row (visible)
// since a config row otherwise doesn't exist until a section type is
// explicitly touched.
func (s *CustomSectionService) CreateSection(ctx context.Context, callerUserID, resumeID string, in repository.CustomSectionInput) (*repository.CustomSection, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	section, err := s.repo.CreateSection(ctx, resumeID, in)
	if err != nil {
		return nil, err
	}
	if err := s.configs.EnsureExists(ctx, resumeID, "custom", &section.ID, in.SortOrder); err != nil {
		return nil, err
	}
	return section, nil
}

func (s *CustomSectionService) UpdateSection(ctx context.Context, callerUserID, resumeID, sectionID string, in repository.CustomSectionInput) (*repository.CustomSection, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	section, err := s.repo.UpdateSection(ctx, resumeID, sectionID, in)
	return section, mapNotFound(err)
}

func (s *CustomSectionService) DeleteSection(ctx context.Context, callerUserID, resumeID, sectionID string) error {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return err
	}
	return mapNotFound(s.repo.DeleteSection(ctx, resumeID, sectionID))
}

func (s *CustomSectionService) ReorderSections(ctx context.Context, callerUserID, resumeID string, orderedIDs []string) error {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return err
	}
	return s.repo.ReorderSections(ctx, resumeID, orderedIDs)
}

func (s *CustomSectionService) CreateEntry(ctx context.Context, callerUserID, resumeID, sectionID string, in repository.CustomSectionEntryInput) (*repository.CustomSectionEntry, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	e, err := s.repo.CreateEntry(ctx, resumeID, sectionID, in)
	return e, mapNotFound(err)
}

func (s *CustomSectionService) UpdateEntry(ctx context.Context, callerUserID, resumeID, sectionID, entryID string, in repository.CustomSectionEntryInput) (*repository.CustomSectionEntry, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	e, err := s.repo.UpdateEntry(ctx, resumeID, sectionID, entryID, in)
	return e, mapNotFound(err)
}

func (s *CustomSectionService) DeleteEntry(ctx context.Context, callerUserID, resumeID, sectionID, entryID string) error {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return err
	}
	return mapNotFound(s.repo.DeleteEntry(ctx, resumeID, sectionID, entryID))
}

func (s *CustomSectionService) ReorderEntries(ctx context.Context, callerUserID, resumeID, sectionID string, orderedIDs []string) error {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return err
	}
	return mapNotFound(s.repo.ReorderEntries(ctx, resumeID, sectionID, orderedIDs))
}
