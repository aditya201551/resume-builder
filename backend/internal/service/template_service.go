package service

import (
	"context"

	"resume-builder/backend/internal/design"
)

type TemplateService struct{}

func NewTemplateService() *TemplateService {
	return &TemplateService{}
}

func (s *TemplateService) List(_ context.Context) ([]design.Template, error) {
	return design.Templates(), nil
}
