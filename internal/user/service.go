package user

import (
	"context"
	"errors"
	"github.com/autobrr/autobrr/internal/domain"
)

type Service interface {
	GetUserCount(ctx context.Context) (int, error)
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	CreateUser(ctx context.Context, user domain.User) error
}

type service struct {
	repo domain.UserRepo
}

func NewService(repo domain.UserRepo) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetUserCount(ctx context.Context) (int, error) {
	return s.repo.GetUserCount(ctx)
}

func (s *service) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *service) CreateUser(ctx context.Context, newUser domain.User) error {
	userCount, err := s.repo.GetUserCount(ctx)
	if err != nil {
		return err
	}

	if userCount > 0 {
		return errors.New("only 1 user account is supported at the moment")
	}

	return s.repo.Store(ctx, newUser)
}
