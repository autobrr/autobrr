package auth

import (
	"errors"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/user"
	"github.com/autobrr/autobrr/pkg/argon2id"
)

type Service interface {
	Login(username, password string) (*domain.User, error)
}

type service struct {
	userSvc user.Service
}

func NewService(userSvc user.Service) Service {
	return &service{
		userSvc: userSvc,
	}
}

func (s *service) Login(username, password string) (*domain.User, error) {
	if username == "" || password == "" {
		return nil, errors.New("bad credentials")
	}

	// find user
	user, err := s.userSvc.FindByUsername(username)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New("bad credentials")
	}

	// compare password from reqest and the saved password
	match, err := argon2id.ComparePasswordAndHash(password, user.Password)
	if err != nil {
		return nil, errors.New("error checking credentials")
	}

	if !match {
		return nil, errors.New("bad credentials")
	}

	return user, nil
}
