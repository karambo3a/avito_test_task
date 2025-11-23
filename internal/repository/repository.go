package repository

import (
	"context"
	"database/sql"

	"github.com/karambo3a/avito_test_task/internal/model"
)

type TeamPostgres interface {
	AddTeam(ctx context.Context, team model.Team) (*model.Team, error)
	GetTeam(ctx context.Context, teamName string) (*model.Team, error)
}

type UsersPostgres interface {
	SetUserIsActive(ctx context.Context, userID string, isActive bool) (*model.User, error)
	GetUserReview(ctx context.Context, userID string) ([]model.PullRequestShort, error)
}

type PullRequestPostgres interface {
	CreatePR(ctx context.Context, pullRequestID, pullRequestName, authorID string) (*model.PullRequest, error)
	MergePR(ctx context.Context, pullRequestID string) (*model.PullRequest, error)
	ReassignPR(ctx context.Context, pullRequestID, oldReviewerID string) (*model.PullRequest, string, error)
}

type StatisticsPostgres interface {
	GetUserStatistics(ctx context.Context, userID string) (*model.UserStatistics, error)
	GetTeamStatistics(ctx context.Context, teamName string) (*model.TeamStatistics, error)
}

type Repository struct {
	TeamPostgres
	UsersPostgres
	PullRequestPostgres
	StatisticsPostgres
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		TeamPostgres:        NewTeamPostgresRepository(db),
		UsersPostgres:       NewUsersPostgresRepository(db),
		PullRequestPostgres: NewPRPostgresRepository(db),
		StatisticsPostgres:  NewStatisticsPostgresRepository(db),
	}
}
