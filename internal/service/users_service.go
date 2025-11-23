package service

import (
	"context"

	"github.com/karambo3a/avito_test_task/internal/model"
	"github.com/karambo3a/avito_test_task/internal/repository"
)

type UsersService struct {
	repository *repository.Repository
}

func NewUsersService(r *repository.Repository) *UsersService {
	return &UsersService{repository: r}
}

func (s *UsersService) SetUserIsActive(ctx context.Context, userID string, isActive bool) (*model.User, error) {
	if userID == "" {
		return nil, model.NewEmptyFieldError("user_id")
	}
	return s.repository.SetUserIsActive(ctx, userID, isActive)
}

func (s *UsersService) GetUserReview(ctx context.Context, userID string) ([]model.PullRequestShort, error) {
	if userID == "" {
		return nil, model.NewEmptyFieldError("user_id")
	}
	return s.repository.GetUserReview(ctx, userID)
}
