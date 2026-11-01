package service

import (
	"context"
	"errors"
	"resume-builder/backend/internal/jobmap"
	"resume-builder/backend/internal/repository"
	"time"
)

type jobStore interface {
	StoreSource(context.Context, string, []jobmap.Job) error
	Get(context.Context, string, string) (jobmap.Job, error)
	Save(context.Context, string, string) error
	Unsave(context.Context, string, string) error
	ReserveRequest(context.Context, string, string, int) (bool, error)
	ClaimSearch(context.Context, string, time.Duration) (bool, error)
	ReleaseSearch(context.Context, string) error
	ProviderRetryAt(context.Context, string) (*time.Time, error)
	DelayFeed(context.Context, string, time.Time) error
	MarkJobClosed(context.Context, string, time.Time) error
}

type JobService struct {
	repo   jobStore
	client *jobmap.Client
	cfg    jobmap.Config
}

func NewJobService(repo jobStore, cfg jobmap.Config) *JobService {
	return &JobService{repo: repo, cfg: cfg, client: jobmap.NewClient(cfg)}
}

func (s *JobService) Get(ctx context.Context, userID, id string) (jobmap.Job, error) {
	if userID == "" {
		return jobmap.Job{}, ErrNotFound
	}
	job, err := s.repo.Get(ctx, userID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return job, ErrNotFound
	}
	if err == nil && job.Source == "himalayas" {
		return s.verifyHimalayas(ctx, userID, job)
	}
	return job, err
}
func (s *JobService) Save(ctx context.Context, userID, id string) error {
	if userID == "" {
		return ErrNotFound
	}
	err := s.repo.Save(ctx, userID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
func (s *JobService) Unsave(ctx context.Context, userID, id string) error {
	if userID == "" {
		return ErrNotFound
	}
	return s.repo.Unsave(ctx, userID, id)
}

func (s *JobService) Countries() []jobmap.Country { return jobmap.Countries() }
