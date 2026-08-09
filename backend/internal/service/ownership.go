package service

import (
	"context"
	"errors"
	"fmt"

	"resume-builder/backend/internal/repository"
)

func ensureResumeOwner(ctx context.Context, resumes *repository.ResumeRepository, resumeID, userID string) (*repository.Resume, error) {
	resume, err := resumes.FindByID(ctx, resumeID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find resume: %w", err)
	}
	if resume.UserID != userID {
		return nil, ErrNotFound
	}
	return resume, nil
}
