package service

import (
	"context"
	"fmt"

	"resume-builder/backend/internal/repository"
)

func NewWorkExperienceService(resumes *repository.ResumeRepository, repo *repository.WorkExperienceRepository) EntityService[repository.WorkExperienceInput, repository.WorkExperience] {
	return newGenericEntityService[repository.WorkExperienceInput, repository.WorkExperience](resumes, repo)
}

func NewEducationService(resumes *repository.ResumeRepository, repo *repository.EducationRepository) EntityService[repository.EducationInput, repository.Education] {
	return newGenericEntityService[repository.EducationInput, repository.Education](resumes, repo)
}

func NewProjectService(resumes *repository.ResumeRepository, repo *repository.ProjectRepository) EntityService[repository.ProjectInput, repository.Project] {
	return newGenericEntityService[repository.ProjectInput, repository.Project](resumes, repo)
}

func NewCertificationService(resumes *repository.ResumeRepository, repo *repository.CertificationRepository) EntityService[repository.CertificationInput, repository.Certification] {
	return newGenericEntityService[repository.CertificationInput, repository.Certification](resumes, repo)
}

func NewLanguageService(resumes *repository.ResumeRepository, repo *repository.LanguageRepository) EntityService[repository.LanguageInput, repository.Language] {
	return newGenericEntityService[repository.LanguageInput, repository.Language](resumes, repo)
}

var validMiscEntryKinds = map[string]bool{"award": true, "publication": true, "volunteer": true}

type miscEntryService struct {
	EntityService[repository.MiscEntryInput, repository.MiscEntry]
}

func NewMiscEntryService(resumes *repository.ResumeRepository, repo *repository.MiscEntryRepository) EntityService[repository.MiscEntryInput, repository.MiscEntry] {
	return &miscEntryService{EntityService: newGenericEntityService[repository.MiscEntryInput, repository.MiscEntry](resumes, repo)}
}

func (s *miscEntryService) Create(ctx context.Context, callerUserID, resumeID string, in repository.MiscEntryInput) (*repository.MiscEntry, error) {
	if !validMiscEntryKinds[in.Kind] {
		return nil, fmt.Errorf("%w: kind must be one of award, publication, volunteer", ErrValidation)
	}
	return s.EntityService.Create(ctx, callerUserID, resumeID, in)
}

func (s *miscEntryService) Update(ctx context.Context, callerUserID, resumeID, id string, in repository.MiscEntryInput) (*repository.MiscEntry, error) {
	if !validMiscEntryKinds[in.Kind] {
		return nil, fmt.Errorf("%w: kind must be one of award, publication, volunteer", ErrValidation)
	}
	return s.EntityService.Update(ctx, callerUserID, resumeID, id, in)
}
