package user

import (
	"context"
	"github.com/autobrr/autobrr/internal/domain"
)

type Service interface {
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
}

type service struct {
	repo domain.UserRepo
}

func NewService(repo domain.UserRepo) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	return user, nil
}
