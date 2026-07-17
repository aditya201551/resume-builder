package service

import (
	"context"

	"resume-builder/backend/internal/repository"
)

// TemplateService has no ownership checks — templates are global, read-only
// reference data, not scoped to any user.
type TemplateService struct {
	repo *repository.TemplateRepository
}

func NewTemplateService(repo *repository.TemplateRepository) *TemplateService {
	return &TemplateService{repo: repo}
}

func (s *TemplateService) List(ctx context.Context) ([]repository.Template, error) {
	return s.repo.ListActive(ctx)
}
