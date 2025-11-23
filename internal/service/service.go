package service

import (
	"context"

	"github.com/karambo3a/avito_test_task/internal/model"
	"github.com/karambo3a/avito_test_task/internal/repository"
)

type Team interface {
	AddTeam(ctx context.Context, team model.Team) (*model.Team, error)
	GetTeam(ctx context.Context, teamName string) (*model.Team, error)
}

type Users interface {
	SetUserIsActive(ctx context.Context, userID string, isActive bool) (*model.User, error)
	GetUserReview(ctx context.Context, userID string) ([]model.PullRequestShort, error)
}

type PullRequest interface {
	CreatePR(ctx context.Context, pullRequestID, pullRequestName, authorID string) (*model.PullRequest, error)
	MergePR(ctx context.Context, pullRequestID string) (*model.PullRequest, error)
	ReassignPR(ctx context.Context, pullRequestID, oldReviewerID string) (*model.PullRequest, string, error)
}

type Statistics interface {
	GetUserStatistics(ctx context.Context, userID string) (*model.UserStatistics, error)
	GetTeamStatistics(ctx context.Context, teamName string) (*model.TeamStatistics, error)
}

type Service struct {
	Team
	Users
	PullRequest
	Statistics
}

func NewService(r *repository.Repository) *Service {
	return &Service{
		Team:        NewTeamService(r),
		Users:       NewUsersService(r),
		PullRequest: NewPullRequestService(r),
		Statistics:  NewStatisticsService(r),
	}
}
