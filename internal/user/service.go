package user

import "github.com/autobrr/autobrr/internal/domain"

type Service interface {
	FindByUsername(username string) (*domain.User, error)
}

type service struct {
	repo domain.UserRepo
}

func NewService(repo domain.UserRepo) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) FindByUsername(username string) (*domain.User, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return nil, err
	}

	return user, nil
}
