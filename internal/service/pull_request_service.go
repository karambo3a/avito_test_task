package service

import (
	"context"

	"github.com/karambo3a/avito_test_task/internal/model"
	"github.com/karambo3a/avito_test_task/internal/repository"
)

type PullRequestService struct {
	repository *repository.Repository
}

func NewPullRequestService(r *repository.Repository) *PullRequestService {
	return &PullRequestService{repository: r}
}

func (s *PullRequestService) CreatePR(ctx context.Context, pullRequestID, pullRequestName, authorID string) (*model.PullRequest, error) {
	switch {
	case pullRequestID == "":
		return nil, model.NewEmptyFieldError("pull_request_id")
	case pullRequestName == "":
		return nil, model.NewEmptyFieldError("pull_request_name")
	case authorID == "":
		return nil, model.NewEmptyFieldError("author_id")
	}
	return s.repository.CreatePR(ctx, pullRequestID, pullRequestName, authorID)
}

func (s *PullRequestService) MergePR(ctx context.Context, pullRequestID string) (*model.PullRequest, error) {
	if pullRequestID == "" {
		return nil, model.NewEmptyFieldError("pull_request_id")
	}
	return s.repository.MergePR(ctx, pullRequestID)
}

func (s *PullRequestService) ReassignPR(ctx context.Context, pullRequestID, oldReviewerID string) (*model.PullRequest, string, error) {
	switch {
	case pullRequestID == "":
		return nil, "", model.NewEmptyFieldError("pull_request_id")
	case oldReviewerID == "":
		return nil, "", model.NewEmptyFieldError("old_reviewer_id")
	}
	return s.repository.ReassignPR(ctx, pullRequestID, oldReviewerID)
}
