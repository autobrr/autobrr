// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/png"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/user"
	"github.com/autobrr/autobrr/pkg/argon2id"

	"github.com/pkg/errors"
	"github.com/pquerna/otp/totp"
	"github.com/rs/zerolog"
)

type Service interface {
	GetUserCount(ctx context.Context) (int, error)
	Login(ctx context.Context, username, password string) (*domain.User, error)
	CreateUser(ctx context.Context, req domain.CreateUserRequest) error
	UpdateUser(ctx context.Context, req domain.UpdateUserRequest) error
	CreateHash(password string) (hash string, err error)
	ComparePasswordAndHash(password string, hash string) (match bool, err error)
	Get2FAStatus(ctx context.Context, username string) (bool, error)
	Enable2FA(ctx context.Context, username string) (string, string, error)
	Verify2FA(ctx context.Context, username string, code string) error
	Verify2FALogin(ctx context.Context, username string, code string) error
	Disable2FA(ctx context.Context, username string) error
}

type service struct {
	log     zerolog.Logger
	userSvc user.Service
}

func NewService(log logger.Logger, userSvc user.Service) Service {
	return &service{
		log:     log.With().Str("module", "auth").Logger(),
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
		s.log.Error().Err(err).Msgf("could not find user by username: %v", username)
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

func (s *service) CreateUser(ctx context.Context, req domain.CreateUserRequest) error {
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
		s.log.Error().Err(err).Msgf("could not create user: %s", req.Username)
		return errors.New("failed to create new user")
	}

	return nil
}

func (s *service) UpdateUser(ctx context.Context, req domain.UpdateUserRequest) error {
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
		s.log.Trace().Err(err).Msgf("invalid login %v", req.UsernameCurrent)
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
		s.log.Debug().Msgf("bad credentials: %q | %q", req.UsernameCurrent, req.PasswordCurrent)
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
		s.log.Error().Err(err).Msgf("could not change password for user: %s", req.UsernameCurrent)
		return errors.New("failed to change password")
	}

	return nil
}

func (s *service) ComparePasswordAndHash(password string, hash string) (match bool, err error) {
	return argon2id.ComparePasswordAndHash(password, hash)
}

func (s *service) CreateHash(password string) (hash string, err error) {
	if password == "" {
		return "", errors.New("must supply non empty password to CreateHash")
	}

	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func (s *service) Get2FAStatus(ctx context.Context, username string) (bool, error) {
	user, err := s.userSvc.FindByUsername(ctx, username)
	if err != nil {
		return false, errors.Wrap(err, "failed to get user")
	}

	return user.TwoFactorAuth, nil
}

func (s *service) Enable2FA(ctx context.Context, username string) (string, string, error) {
	// Generate QR code
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "autobrr",
		AccountName: username,
		SecretSize:  20,
	})
	if err != nil {
		return "", "", errors.Wrap(err, "failed to generate TOTP key")
	}

	// Generate QR code image
	var buf bytes.Buffer
	img, err := key.Image(200, 200)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to generate QR code image")
	}

	if err := png.Encode(&buf, img); err != nil {
		return "", "", errors.Wrap(err, "failed to encode QR code image")
	}

	// Convert to base64
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	// Store secret in database but don't enable 2FA yet
	if err := s.userSvc.Store2FASecret(ctx, username, key.Secret()); err != nil {
		return "", "", errors.Wrap(err, "failed to store 2FA secret")
	}

	s.log.Debug().
		Str("username", username).
		Str("secret", key.Secret()).
		Msg("stored 2FA secret")

	return dataURL, key.Secret(), nil
}

// Verify2FA verifies the 2FA code during setup
func (s *service) Verify2FA(ctx context.Context, username string, code string) error {
	// Get user's secret
	secret, err := s.userSvc.Get2FASecret(ctx, username)
	if err != nil {
		return errors.Wrap(err, "failed to get 2FA secret")
	}

	s.log.Debug(). // unsure if this is a helpful log or not
			Str("username", username).
			Str("code", code).
			Str("secret", secret).
			Msg("attempting 2FA verification during setup")

	// Generate current valid codes for debugging
	validCodes := make([]string, 3)
	now := time.Now()
	for i := -1; i <= 1; i++ {
		if validCode, err := totp.GenerateCode(secret, now.Add(30*time.Duration(i)*time.Second)); err == nil {
			validCodes[i+1] = validCode
		}
	}

	s.log.Debug(). // unsure if this is a helpful log or not
			Str("username", username).
			Strs("valid_codes", validCodes).
			Msg("valid codes for current time window")

	// Validate the code with a wider window during setup
	valid := totp.Validate(code, secret)

	if !valid {
		s.log.Debug().
			Str("username", username).
			Str("code", code).
			Msg("invalid 2FA code during setup")
		return errors.New("invalid verification code")
	}

	// Enable 2FA after successful verification
	if err := s.userSvc.Enable2FA(ctx, username, secret); err != nil {
		return errors.Wrap(err, "failed to enable 2FA")
	}

	return nil
}

// Verify2FALogin verifies the 2FA code during login
func (s *service) Verify2FALogin(ctx context.Context, username string, code string) error {
	// Get user's secret
	secret, err := s.userSvc.Get2FASecret(ctx, username)
	if err != nil {
		return errors.Wrap(err, "failed to get 2FA secret")
	}

	s.log.Debug().
		Str("username", username).
		Str("code", code).
		Str("secret", secret).
		Msg("attempting 2FA login verification")

	// Validate the code
	valid := totp.Validate(code, secret)

	if !valid {
		s.log.Debug().
			Str("username", username).
			Str("code", code).
			Msg("invalid 2FA code during login")
		return errors.New("invalid verification code")
	}

	return nil
}

func (s *service) Disable2FA(ctx context.Context, username string) error {
	if err := s.userSvc.Disable2FA(ctx, username); err != nil {
		return errors.Wrap(err, "failed to disable 2FA")
	}

	return nil
}
