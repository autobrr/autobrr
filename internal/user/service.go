// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package user

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
)

type Service interface {
	GetUserCount(ctx context.Context) (int, error)
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	CreateUser(ctx context.Context, req domain.CreateUserRequest) error
	Update(ctx context.Context, req domain.UpdateUserRequest) error
	Delete(ctx context.Context, username string) error
	Enable2FA(ctx context.Context, username string, secret string) error
	Store2FASecret(ctx context.Context, username string, secret string) error
	Get2FASecret(ctx context.Context, username string) (string, error)
	Disable2FA(ctx context.Context, username string) error
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
	return s.repo.FindByUsername(ctx, username)
}

func (s *service) CreateUser(ctx context.Context, req domain.CreateUserRequest) error {
	return s.repo.Store(ctx, req)
}

func (s *service) Update(ctx context.Context, req domain.UpdateUserRequest) error {
	return s.repo.Update(ctx, req)
}

func (s *service) Delete(ctx context.Context, username string) error {
	return s.repo.Delete(ctx, username)
}

func (s *service) Enable2FA(ctx context.Context, username string, secret string) error {
	return s.repo.Enable2FA(ctx, username, secret)
}

func (s *service) Store2FASecret(ctx context.Context, username string, secret string) error {
	return s.repo.Store2FASecret(ctx, username, secret)
}

func (s *service) Get2FASecret(ctx context.Context, username string) (string, error) {
	return s.repo.Get2FASecret(ctx, username)
}

func (s *service) Disable2FA(ctx context.Context, username string) error {
	return s.repo.Disable2FA(ctx, username)
}
