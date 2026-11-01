package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"resume-builder/backend/internal/jobsearch"
)

type jobPlanner interface {
	Refine(context.Context, jobsearch.Intent, string, []jobsearch.Turn) (jobsearch.Refinement, error)
}
type jobAssistantBudget interface {
	ReserveRequest(context.Context, string, string, int) (bool, error)
}
type JobAssistantService struct {
	planner jobPlanner
	repo    jobAssistantBudget
}

func NewJobAssistantService(planner jobPlanner, repo jobAssistantBudget) *JobAssistantService {
	return &JobAssistantService{planner: planner, repo: repo}
}

func (s *JobAssistantService) Refine(ctx context.Context, userID string, intent jobsearch.Intent, message string, history []jobsearch.Turn) (jobsearch.Refinement, error) {
	if userID == "" {
		return jobsearch.Refinement{}, ErrNotFound
	}
	if err := intent.Validate(); err != nil {
		return jobsearch.Refinement{}, fmt.Errorf("%s: %w", err, ErrValidation)
	}
	message = strings.TrimSpace(message)
	if len(history) > 4 {
		return jobsearch.Refinement{}, ErrValidation
	}
	for _, turn := range history {
		if (turn.Role != "user" && turn.Role != "assistant") || len([]rune(turn.Text)) > 2000 {
			return jobsearch.Refinement{}, ErrValidation
		}
	}
	if message == "" || len([]rune(message)) > 2000 {
		return jobsearch.Refinement{}, fmt.Errorf("describe your search in up to 2000 characters: %w", ErrValidation)
	}
	allowed, err := s.repo.ReserveRequest(ctx, "job-assistant:"+userID, time.Now().UTC().Format("2006-01-02"), 30)
	if err != nil {
		return jobsearch.Refinement{}, err
	}
	if !allowed {
		return jobsearch.Refinement{Intent: intent, Message: "Today's assistant allowance has been used. You can continue searching with the filters.", Apply: false}, nil
	}
	work, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	return s.planner.Refine(work, intent, message, history)
}
