// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
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
	Update(ctx context.Context, req domain.UpdateUserRequest) error
	Enable2FA(ctx context.Context, username string, secret string) error
	Verify2FA(ctx context.Context, username string, code string) error
	Disable2FA(ctx context.Context, username string) error
	Get2FASecret(ctx context.Context, username string) (string, error)
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

func (s *service) Update(ctx context.Context, req domain.UpdateUserRequest) error {
	return s.repo.Update(ctx, req)
}

func (s *service) Enable2FA(ctx context.Context, username string, secret string) error {
	return s.repo.Enable2FA(ctx, username, secret)
}

func (s *service) Verify2FA(ctx context.Context, username string, code string) error {
	return s.repo.Verify2FA(ctx, username, code)
}

func (s *service) Disable2FA(ctx context.Context, username string) error {
	return s.repo.Disable2FA(ctx, username)
}

func (s *service) Get2FASecret(ctx context.Context, username string) (string, error) {
	return s.repo.Get2FASecret(ctx, username)
}
