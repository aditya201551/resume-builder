package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"resume-builder/backend/internal/design"
	"resume-builder/backend/internal/repository"
)

func resolveResumeDesign(
	ctx context.Context,
	designs *repository.ResumeDesignRepository,
	resumeID string,
) (design.ResumeDesign, error) {
	row, err := designs.Find(ctx, resumeID)
	if err == nil {
		d, uerr := design.Decode(row.Design)
		if uerr != nil {
			return design.ResumeDesign{}, fmt.Errorf("unmarshal resume design: %w", uerr)
		}
		return d, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return design.ResumeDesign{}, err
	}

	return design.Default(design.DefaultTemplateID()), nil
}

type ResumeDesignService struct {
	resumes *repository.ResumeRepository
	designs *repository.ResumeDesignRepository
}

func NewResumeDesignService(
	resumes *repository.ResumeRepository,
	designs *repository.ResumeDesignRepository,
) *ResumeDesignService {
	return &ResumeDesignService{resumes: resumes, designs: designs}
}

func (s *ResumeDesignService) Get(ctx context.Context, callerUserID, resumeID string) (*design.ResumeDesign, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	d, err := resolveResumeDesign(ctx, s.designs, resumeID)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *ResumeDesignService) Update(ctx context.Context, callerUserID, resumeID string, in design.ResumeDesign) (*design.ResumeDesign, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	if err := design.Validate(in); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := design.ValidateTemplate(in); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	raw, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("marshal resume design: %w", err)
	}
	if _, err := s.designs.Upsert(ctx, resumeID, raw); err != nil {
		return nil, err
	}
	return &in, nil
}
