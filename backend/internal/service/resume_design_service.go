package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"resume-builder/backend/internal/design"
	"resume-builder/backend/internal/repository"
)

// defaultTemplateRendererKey is the fallback template a resume resolves to
// when it has no resume_designs row yet — see resolveResumeDesign.
const defaultTemplateRendererKey = "classic"

// resolveResumeDesign returns a resume's stored design, or — if it has never
// saved one — the default template's preset with TemplateID filled in. This
// is shared by ResumeDesignService.Get (the dedicated endpoint) and
// ResumeService.GetFullResume (which embeds Design in the aggregate), so
// both ever have exactly one definition of "what a resume's design is."
func resolveResumeDesign(
	ctx context.Context,
	designs *repository.ResumeDesignRepository,
	templates *repository.TemplateRepository,
	resumeID string,
) (design.ResumeDesign, error) {
	row, err := designs.Find(ctx, resumeID)
	if err == nil {
		var d design.ResumeDesign
		if uerr := json.Unmarshal(row.Design, &d); uerr != nil {
			return design.ResumeDesign{}, fmt.Errorf("unmarshal resume design: %w", uerr)
		}
		return d, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return design.ResumeDesign{}, err
	}

	tmpl, err := templates.FindByRendererKey(ctx, defaultTemplateRendererKey)
	if err != nil {
		return design.ResumeDesign{}, fmt.Errorf("find default template: %w", err)
	}
	var d design.ResumeDesign
	if err := json.Unmarshal(tmpl.DefaultDesign, &d); err != nil {
		return design.ResumeDesign{}, fmt.Errorf("unmarshal default template design: %w", err)
	}
	d.TemplateID = tmpl.ID
	return d, nil
}

type ResumeDesignService struct {
	resumes   *repository.ResumeRepository
	designs   *repository.ResumeDesignRepository
	templates *repository.TemplateRepository
}

func NewResumeDesignService(
	resumes *repository.ResumeRepository,
	designs *repository.ResumeDesignRepository,
	templates *repository.TemplateRepository,
) *ResumeDesignService {
	return &ResumeDesignService{resumes: resumes, designs: designs, templates: templates}
}

func (s *ResumeDesignService) Get(ctx context.Context, callerUserID, resumeID string) (*design.ResumeDesign, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	d, err := resolveResumeDesign(ctx, s.designs, s.templates, resumeID)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// Update replaces a resume's design wholesale — same PUT semantics as
// ResumeRepository.UpdateMeta, not a patch. in must already be a complete,
// valid ResumeDesign; Validate is the single gate a future
// propose_design_update agent tool would also call before this.
func (s *ResumeDesignService) Update(ctx context.Context, callerUserID, resumeID string, in design.ResumeDesign) (*design.ResumeDesign, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	if err := design.Validate(in); err != nil {
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
