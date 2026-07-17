package service

import (
	"context"

	"resume-builder/backend/internal/repository"
)

type SectionConfigService struct {
	resumes *repository.ResumeRepository
	repo    *repository.SectionConfigRepository
}

func NewSectionConfigService(resumes *repository.ResumeRepository, repo *repository.SectionConfigRepository) *SectionConfigService {
	return &SectionConfigService{resumes: resumes, repo: repo}
}

func (s *SectionConfigService) List(ctx context.Context, callerUserID, resumeID string) ([]repository.SectionConfig, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, resumeID)
}

func (s *SectionConfigService) Update(ctx context.Context, callerUserID, resumeID, sectionType string, customSectionID *string, in repository.SectionConfigInput) (*repository.SectionConfig, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	c, err := s.repo.Update(ctx, resumeID, sectionType, customSectionID, in)
	return c, mapNotFound(err)
}
