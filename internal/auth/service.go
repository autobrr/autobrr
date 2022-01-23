package auth

import (
	"context"
	"errors"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/user"
	"github.com/autobrr/autobrr/pkg/argon2id"
)

type Service interface {
	Login(ctx context.Context, username, password string) (*domain.User, error)
}

type service struct {
	userSvc user.Service
}

func NewService(userSvc user.Service) Service {
	return &service{
		userSvc: userSvc,
	}
}

func (s *service) Login(ctx context.Context, username, password string) (*domain.User, error) {
	if username == "" || password == "" {
		return nil, errors.New("bad credentials")
	}

	// find user
	u, err := s.userSvc.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	if u == nil {
		return nil, errors.New("bad credentials")
	}

	// compare password from request and the saved password
	match, err := argon2id.ComparePasswordAndHash(password, u.Password)
	if err != nil {
		return nil, errors.New("error checking credentials")
	}

	if !match {
		return nil, errors.New("bad credentials")
	}

	return u, nil
}
