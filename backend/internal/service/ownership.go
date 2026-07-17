package service

import (
	"context"
	"errors"
	"fmt"

	"resume-builder/backend/internal/repository"
)

// ensureResumeOwner confirms resumeID exists and belongs to userID, returning
// the resume itself so callers that need it (e.g. to read template_id) don't
// have to fetch it twice. Every section-scoped service call goes through this
// before touching child rows, so ownership enforcement lives in exactly one
// place regardless of which adapter (REST today, MCP later) calls in.
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
