package service

import (
	"context"

	"resume-builder/backend/internal/repository"
)

// TemplateService has no ownership check — templates aren't resume-scoped,
// they're a shared, global catalog every user reads the same rows from.
type TemplateService struct {
	repo *repository.TemplateRepository
}

func NewTemplateService(repo *repository.TemplateRepository) *TemplateService {
	return &TemplateService{repo: repo}
}

func (s *TemplateService) List(ctx context.Context) ([]repository.Template, error) {
	return s.repo.List(ctx)
}
