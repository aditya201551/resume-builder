package service

import (
	"context"
	"errors"

	"resume-builder/backend/internal/repository"
)

type EntityService[TInput any, TOutput any] interface {
	List(ctx context.Context, callerUserID, resumeID string) ([]TOutput, error)
	Create(ctx context.Context, callerUserID, resumeID string, in TInput) (*TOutput, error)
	Update(ctx context.Context, callerUserID, resumeID, id string, in TInput) (*TOutput, error)
	Delete(ctx context.Context, callerUserID, resumeID, id string) error
	Reorder(ctx context.Context, callerUserID, resumeID string, orderedIDs []string) error
}

type entityRepo[TInput any, TOutput any] interface {
	List(ctx context.Context, resumeID string) ([]TOutput, error)
	Create(ctx context.Context, resumeID string, in TInput) (*TOutput, error)
	Update(ctx context.Context, resumeID, id string, in TInput) (*TOutput, error)
	Delete(ctx context.Context, resumeID, id string) error
	Reorder(ctx context.Context, resumeID string, orderedIDs []string) error
}

type genericEntityService[TInput any, TOutput any] struct {
	resumes *repository.ResumeRepository
	repo    entityRepo[TInput, TOutput]
}

func newGenericEntityService[TInput any, TOutput any](resumes *repository.ResumeRepository, repo entityRepo[TInput, TOutput]) EntityService[TInput, TOutput] {
	return &genericEntityService[TInput, TOutput]{resumes: resumes, repo: repo}
}

func (s *genericEntityService[TInput, TOutput]) List(ctx context.Context, callerUserID, resumeID string) ([]TOutput, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, resumeID)
}

func (s *genericEntityService[TInput, TOutput]) Create(ctx context.Context, callerUserID, resumeID string, in TInput) (*TOutput, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, resumeID, in)
}

func (s *genericEntityService[TInput, TOutput]) Update(ctx context.Context, callerUserID, resumeID, id string, in TInput) (*TOutput, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	out, err := s.repo.Update(ctx, resumeID, id, in)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return out, nil
}

func (s *genericEntityService[TInput, TOutput]) Delete(ctx context.Context, callerUserID, resumeID, id string) error {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return err
	}
	return mapNotFound(s.repo.Delete(ctx, resumeID, id))
}

func (s *genericEntityService[TInput, TOutput]) Reorder(ctx context.Context, callerUserID, resumeID string, orderedIDs []string) error {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return err
	}
	return s.repo.Reorder(ctx, resumeID, orderedIDs)
}

func mapNotFound(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
