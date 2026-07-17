package service

import (
	"context"
	"fmt"

	"resume-builder/backend/internal/repository"
)

var validContentBlockKinds = map[string]bool{"work_experience": true, "project": true}

type ContentBlockService struct {
	resumes *repository.ResumeRepository
	repo    *repository.ContentBlockRepository
}

func NewContentBlockService(resumes *repository.ResumeRepository, repo *repository.ContentBlockRepository) *ContentBlockService {
	return &ContentBlockService{resumes: resumes, repo: repo}
}

func (s *ContentBlockService) List(ctx context.Context, callerUserID, resumeID string) ([]repository.ContentBlock, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, resumeID)
}

// UpdateContent is the single write path Phase 2's AI rewrite tool and the
// frontend's rich-text editor both go through — same validation and
// ownership check regardless of which one calls it.
func (s *ContentBlockService) UpdateContent(ctx context.Context, callerUserID, resumeID, kind, blockID, content string) error {
	if !validContentBlockKinds[kind] {
		return fmt.Errorf("%w: kind must be one of work_experience, project", ErrValidation)
	}
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return err
	}
	return mapNotFound(s.repo.UpdateContent(ctx, resumeID, kind, blockID, content))
}
