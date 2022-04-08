package auth

import (
	"context"
	"errors"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/user"
	"github.com/autobrr/autobrr/pkg/argon2id"
)

type Service interface {
	GetUserCount(ctx context.Context) (int, error)
	Login(ctx context.Context, username, password string) (*domain.User, error)
	CreateUser(ctx context.Context, username, password string) error
}

type service struct {
	userSvc user.Service
}

func NewService(userSvc user.Service) Service {
	return &service{
		userSvc: userSvc,
	}
}

func (s *service) GetUserCount(ctx context.Context) (int, error) {
	return s.userSvc.GetUserCount(ctx)
}

func (s *service) Login(ctx context.Context, username, password string) (*domain.User, error) {
	if username == "" || password == "" {
		return nil, errors.New("empty credentials supplied")
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

func (s *service) CreateUser(ctx context.Context, username, password string) error {
	if username == "" || password == "" {
		return errors.New("empty credentials supplied")
	}

	userCount, err := s.userSvc.GetUserCount(ctx)
	if err != nil {
		return err
	}

	if userCount > 0 {
		return errors.New("only 1 user account is supported at the moment")
	}

	hashed, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return errors.New("failed to hash password")
	}

	newUser := domain.User{
		Username: username,
		Password: hashed,
	}
	if err := s.userSvc.CreateUser(context.Background(), newUser); err != nil {
		return errors.New("failed to create new user")
	}

	return nil
}
