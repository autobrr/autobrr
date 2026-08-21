// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package auth

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/argon2id"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type userService interface {
	GetUserCount(ctx context.Context) (int, error)
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	CreateUser(ctx context.Context, req domain.CreateUserRequest) error
	Update(ctx context.Context, req domain.UpdateUserRequest) error
}

type Service struct {
	log     zerolog.Logger
	userSvc userService
}

func NewService(log zerolog.Logger, userSvc userService) *Service {
	return &Service{
		log:     log.With().Str("module", "auth").Logger(),
		userSvc: userSvc,
	}
}

func (s *Service) GetUserCount(ctx context.Context) (int, error) {
	return s.userSvc.GetUserCount(ctx)
}

func (s *Service) Login(ctx context.Context, username, password string) (*domain.User, error) {
	if username == "" || password == "" {
		return nil, errors.New("empty credentials supplied")
	}

	// find user
	u, err := s.userSvc.FindByUsername(ctx, username)
	if err != nil {
		s.log.Error().Err(err).Str("username", username).Msg("could not find user by username")
		return nil, errors.Wrapf(err, "invalid login: %s", username)
	}

	if u == nil {
		return nil, errors.Errorf("invalid login: %s", username)
	}

	// compare password from request and the saved password
	match, err := s.ComparePasswordAndHash(password, u.Password)
	if err != nil {
		return nil, errors.New("error checking credentials")
	}

	if !match {
		s.log.Error().Msg("bad credentials")
		return nil, errors.Errorf("invalid login: %s", username)
	}

	return u, nil
}

func (s *Service) CreateUser(ctx context.Context, req domain.CreateUserRequest) error {
	if req.Username == "" {
		return errors.New("validation error: empty username supplied")
	} else if req.Password == "" {
		return errors.New("validation error: empty password supplied")
	}

	userCount, err := s.userSvc.GetUserCount(ctx)
	if err != nil {
		return err
	}

	if userCount > 0 {
		return errors.New("only 1 user account is supported at the moment")
	}

	hashed, err := s.CreateHash(req.Password)
	if err != nil {
		return errors.New("failed to hash password")
	}

	req.Password = hashed

	if err := s.userSvc.CreateUser(ctx, req); err != nil {
		s.log.Error().Err(err).Str("username", req.Username).Msg("could not create user")
		return errors.New("failed to create new user")
	}

	return nil
}

func (s *Service) UpdateUser(ctx context.Context, req domain.UpdateUserRequest) error {
	if req.PasswordCurrent == "" {
		return errors.New("validation error: empty current password supplied")
	}

	if req.PasswordNew != "" && req.PasswordCurrent != "" {
		if req.PasswordNew == req.PasswordCurrent {
			return errors.New("validation error: new password must be different")
		}
	}

	// find user
	u, err := s.userSvc.FindByUsername(ctx, req.UsernameCurrent)
	if err != nil {
		s.log.Trace().Err(err).Str("username", req.UsernameCurrent).Msg("invalid login")
		return errors.Wrapf(err, "invalid login: %s", req.UsernameCurrent)
	}

	if u == nil {
		return errors.Errorf("invalid login: %s", req.UsernameCurrent)
	}

	// compare password from request and the saved password
	match, err := s.ComparePasswordAndHash(req.PasswordCurrent, u.Password)
	if err != nil {
		return errors.New("error checking credentials")
	}

	if !match {
		s.log.Debug().Str("username", req.UsernameCurrent).Msg("bad credentials")
		return errors.Errorf("invalid login: %s", req.UsernameCurrent)
	}

	if req.PasswordNew != "" {
		hashed, err := s.CreateHash(req.PasswordNew)
		if err != nil {
			return errors.New("failed to hash password")
		}

		req.PasswordNewHash = hashed
	}

	if err := s.userSvc.Update(ctx, req); err != nil {
		s.log.Error().Err(err).Str("username", req.UsernameCurrent).Msg("could not change password for user")
		return errors.New("failed to change password")
	}

	return nil
}

func (s *Service) ComparePasswordAndHash(password string, hash string) (match bool, err error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func (s *Service) CreateHash(password string) (hash string, err error) {
	if password == "" {
		return "", errors.New("must supply non empty password to CreateHash")
	}

	return argon2id.CreateHash(password, argon2id.DefaultParams)
}
