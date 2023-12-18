// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package user

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
)

type Service interface {
	GetUserCount(ctx context.Context) (int, error)
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	CreateUser(ctx context.Context, req domain.CreateUserRequest) error
	ChangePasswordByUsername(ctx context.Context, req domain.ChangePasswordRequest) error
	ChangeUsername(ctx context.Context, req domain.ChangeUsernameRequest) error
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

func (s *service) CreateUser(ctx context.Context, req domain.CreateUserRequest) error {
	userCount, err := s.repo.GetUserCount(ctx)
	if err != nil {
		return err
	}

	if userCount > 0 {
		return errors.New("only 1 user account is supported at the moment")
	}

	return s.repo.Store(ctx, req)
}

func (s *service) updateCredentials(ctx context.Context, username string, updateFunc func(*domain.User)) error {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return err
	}

	updateFunc(user)

	return s.repo.Update(ctx, *user)
}

func (s *service) ChangePasswordByUsername(ctx context.Context, req domain.ChangePasswordRequest) error {
	return s.updateCredentials(ctx, req.Username, func(user *domain.User) {
		user.Password = req.NewPassword
	})
}

func (s *service) ChangeUsername(ctx context.Context, req domain.ChangeUsernameRequest) error {
	return s.updateCredentials(ctx, req.Username, func(user *domain.User) {
		user.Username = req.NewUsername
	})
}
