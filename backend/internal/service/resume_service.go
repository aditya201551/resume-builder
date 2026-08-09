package service

import (
	"context"

	"resume-builder/backend/internal/design"
	"resume-builder/backend/internal/repository"
)

type FullResume struct {
	Resume          repository.Resume           `json:"resume"`
	Design          design.ResumeDesign         `json:"design"`
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
	resumeDesigns   *repository.ResumeDesignRepository
	templates       *repository.TemplateRepository
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
	resumeDesigns *repository.ResumeDesignRepository,
	templates *repository.TemplateRepository,
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
		resumeDesigns:   resumeDesigns,
		templates:       templates,
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
	resumeDesign, err := resolveResumeDesign(ctx, s.resumeDesigns, s.templates, resumeID)
	if err != nil {
		return nil, err
	}

	return &FullResume{
		Resume:          *resume,
		Design:          resumeDesign,
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
