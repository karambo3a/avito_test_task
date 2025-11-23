package service

import (
	"context"

	"github.com/karambo3a/avito_test_task/internal/model"
	"github.com/karambo3a/avito_test_task/internal/repository"
)

type StatisticsService struct {
	repository *repository.Repository
}

func NewStatisticsService(repo *repository.Repository) *StatisticsService {
	return &StatisticsService{repository: repo}
}

func (s *StatisticsService) GetUserStatistics(ctx context.Context, userID string) (*model.UserStatistics, error) {
	if userID == "" {
		return nil, model.NewEmptyFieldError("user_id")
	}

	return s.repository.GetUserStatistics(ctx, userID)
}

func (s *StatisticsService) GetTeamStatistics(ctx context.Context, teamName string) (*model.TeamStatistics, error) {
	if teamName == "" {
		return nil, model.NewEmptyFieldError("team_id")
	}

	return s.repository.GetTeamStatistics(ctx, teamName)
}
