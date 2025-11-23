package service

import (
	"context"

	"github.com/karambo3a/avito_test_task/internal/model"
	"github.com/karambo3a/avito_test_task/internal/repository"
)

type TeamService struct {
	repository *repository.Repository
}

func NewTeamService(r *repository.Repository) *TeamService {
	return &TeamService{repository: r}
}

func (s *TeamService) AddTeam(ctx context.Context, team model.Team) (*model.Team, error) {
	if team.TeamName == "" {
		return nil, model.NewEmptyFieldError("team_name")
	}
	if len(team.Members) == 0 {
		return nil, model.NewEmptyFieldError("members")
	}

	for _, member := range team.Members {
		if member.UserID == "" {
			return nil, model.NewEmptyFieldError("user_id")
		}
	}
	return s.repository.AddTeam(ctx, team)
}

func (s *TeamService) GetTeam(ctx context.Context, teamName string) (*model.Team, error) {
	if teamName == "" {
		return nil, model.NewEmptyFieldError("team_name")
	}
	return s.repository.GetTeam(ctx, teamName)
}
