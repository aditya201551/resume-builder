package service

import (
	"context"

	"resume-builder/backend/internal/repository"
)

type TemplateService struct {
	repo *repository.TemplateRepository
}

func NewTemplateService(repo *repository.TemplateRepository) *TemplateService {
	return &TemplateService{repo: repo}
}

func (s *TemplateService) List(ctx context.Context) ([]repository.Template, error) {
	return s.repo.List(ctx)
}
