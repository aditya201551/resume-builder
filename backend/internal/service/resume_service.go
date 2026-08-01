package service

import (
	"context"

	"resume-builder/backend/internal/repository"
)

// FullResume is the aggregate read the frontend doesn't need today (it
// renders from local state) but a Phase 2+ agent will, since it has no
// browser-side state to read from — one call returns everything needed to
// reason about or export the resume.
type FullResume struct {
	Resume          repository.Resume           `json:"resume"`
	WorkExperiences []repository.WorkExperience `json:"work_experiences"`
	Educations      []repository.Education      `json:"educations"`
	SkillGroups     []repository.SkillGroup     `json:"skill_groups"`
	Projects        []repository.Project        `json:"projects"`
	Certifications  []repository.Certification  `json:"certifications"`
	Languages       []repository.Language       `json:"languages"`
	MiscEntries     []repository.MiscEntry      `json:"misc_entries"`
	CustomSections  []repository.CustomSection  `json:"custom_sections"`
	SectionConfigs  []repository.SectionConfig  `json:"section_configs"`
}

type ResumeService struct {
	resumes         *repository.ResumeRepository
	workExperiences *repository.WorkExperienceRepository
	educations      *repository.EducationRepository
	skills          *repository.SkillRepository
	projects        *repository.ProjectRepository
	certifications  *repository.CertificationRepository
	languages       *repository.LanguageRepository
	miscEntries     *repository.MiscEntryRepository
	customSections  *repository.CustomSectionRepository
	sectionConfigs  *repository.SectionConfigRepository
}

func NewResumeService(
	resumes *repository.ResumeRepository,
	workExperiences *repository.WorkExperienceRepository,
	educations *repository.EducationRepository,
	skills *repository.SkillRepository,
	projects *repository.ProjectRepository,
	certifications *repository.CertificationRepository,
	languages *repository.LanguageRepository,
	miscEntries *repository.MiscEntryRepository,
	customSections *repository.CustomSectionRepository,
	sectionConfigs *repository.SectionConfigRepository,
) *ResumeService {
	return &ResumeService{
		resumes:         resumes,
		workExperiences: workExperiences,
		educations:      educations,
		skills:          skills,
		projects:        projects,
		certifications:  certifications,
		languages:       languages,
		miscEntries:     miscEntries,
		customSections:  customSections,
		sectionConfigs:  sectionConfigs,
	}
}

func (s *ResumeService) Create(ctx context.Context, userID string, in repository.ResumeMetaInput) (*repository.Resume, error) {
	return s.resumes.Create(ctx, userID, in)
}

func (s *ResumeService) Get(ctx context.Context, callerUserID, resumeID string) (*repository.Resume, error) {
	return ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID)
}

func (s *ResumeService) List(ctx context.Context, userID string) ([]repository.Resume, error) {
	return s.resumes.ListByUser(ctx, userID)
}

func (s *ResumeService) UpdateMeta(ctx context.Context, callerUserID, resumeID string, in repository.ResumeMetaInput) (*repository.Resume, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	return s.resumes.UpdateMeta(ctx, resumeID, in)
}

func (s *ResumeService) Delete(ctx context.Context, callerUserID, resumeID string) error {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return err
	}
	return s.resumes.Delete(ctx, resumeID)
}

func (s *ResumeService) Duplicate(ctx context.Context, callerUserID, resumeID string) (*repository.Resume, error) {
	if _, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID); err != nil {
		return nil, err
	}
	return s.resumes.Duplicate(ctx, resumeID)
}

// GetFullResume assembles the resume and every child collection into one
// document — the shape both a "load for editing" call and a future MCP
// read-only tool want, so it lives here once rather than being reinvented
// per adapter.
func (s *ResumeService) GetFullResume(ctx context.Context, callerUserID, resumeID string) (*FullResume, error) {
	resume, err := ensureResumeOwner(ctx, s.resumes, resumeID, callerUserID)
	if err != nil {
		return nil, err
	}

	workExperiences, err := s.workExperiences.List(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	educations, err := s.educations.List(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	skillGroups, err := s.skills.ListGroups(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	projects, err := s.projects.List(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	certifications, err := s.certifications.List(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	languages, err := s.languages.List(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	miscEntries, err := s.miscEntries.List(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	customSections, err := s.customSections.ListSections(ctx, resumeID)
	if err != nil {
		return nil, err
	}
	sectionConfigs, err := s.sectionConfigs.List(ctx, resumeID)
	if err != nil {
		return nil, err
	}

	return &FullResume{
		Resume:          *resume,
		WorkExperiences: workExperiences,
		Educations:      educations,
		SkillGroups:     skillGroups,
		Projects:        projects,
		Certifications:  certifications,
		Languages:       languages,
		MiscEntries:     miscEntries,
		CustomSections:  customSections,
		SectionConfigs:  sectionConfigs,
	}, nil
}
