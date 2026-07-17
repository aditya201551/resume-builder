package service

import (
	"context"

	"resume-builder/backend/internal/repository"
)

type SkillService struct {
	resumes *repository.ResumeRepository
	repo    *repository.SkillRepository
}

func NewSkillService(resumes *repository.ResumeRepository, repo *repository.SkillRepository) *SkillService {
	return &SkillService{resumes: resumes, repo: repo}
}

func (s *SkillService) ListGroups(ctx context.Context, callerUserID, resumeID string) ([]repository.SkillGroup, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	return s.repo.ListGroups(ctx, resumeID)
}

func (s *SkillService) CreateGroup(ctx context.Context, callerUserID, resumeID string, in repository.SkillGroupInput) (*repository.SkillGroup, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	return s.repo.CreateGroup(ctx, resumeID, in)
}

func (s *SkillService) UpdateGroup(ctx context.Context, callerUserID, resumeID, groupID string, in repository.SkillGroupInput) (*repository.SkillGroup, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	g, err := s.repo.UpdateGroup(ctx, resumeID, groupID, in)
	return g, mapNotFound(err)
}

func (s *SkillService) DeleteGroup(ctx context.Context, callerUserID, resumeID, groupID string) error {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return err
	}
	return mapNotFound(s.repo.DeleteGroup(ctx, resumeID, groupID))
}

func (s *SkillService) ReorderGroups(ctx context.Context, callerUserID, resumeID string, orderedIDs []string) error {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return err
	}
	return s.repo.ReorderGroups(ctx, resumeID, orderedIDs)
}

func (s *SkillService) CreateItem(ctx context.Context, callerUserID, resumeID, groupID string, in repository.SkillItemInput) (*repository.SkillItem, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	it, err := s.repo.CreateItem(ctx, resumeID, groupID, in)
	return it, mapNotFound(err)
}

func (s *SkillService) UpdateItem(ctx context.Context, callerUserID, resumeID, groupID, itemID string, in repository.SkillItemInput) (*repository.SkillItem, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	it, err := s.repo.UpdateItem(ctx, resumeID, groupID, itemID, in)
	return it, mapNotFound(err)
}

func (s *SkillService) DeleteItem(ctx context.Context, callerUserID, resumeID, groupID, itemID string) error {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return err
	}
	return mapNotFound(s.repo.DeleteItem(ctx, resumeID, groupID, itemID))
}

func (s *SkillService) ReorderItems(ctx context.Context, callerUserID, resumeID, groupID string, orderedIDs []string) error {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return err
	}
	return mapNotFound(s.repo.ReorderItems(ctx, resumeID, groupID, orderedIDs))
}
