// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/alphadose/haxmap"
	"github.com/rs/zerolog"
)

type repo interface {
	Store(ctx context.Context, key *domain.APIKey) error
	Update(ctx context.Context, key *domain.APIKey) error
	Delete(ctx context.Context, key string) error
	GetAllAPIKeys(ctx context.Context) ([]domain.APIKey, error)
	GetKey(ctx context.Context, key string) (*domain.APIKey, error)
}

type Service struct {
	log  zerolog.Logger
	repo repo

	keyCache *haxmap.Map[string, domain.APIKey]
}

func NewService(log zerolog.Logger, repo repo) *Service {
	return &Service{
		log:      log.With().Str("module", "api").Logger(),
		repo:     repo,
		keyCache: haxmap.New[string, domain.APIKey](),
	}
}

func (s *Service) List(ctx context.Context) ([]domain.APIKey, error) {
	return s.repo.GetAllAPIKeys(ctx)
}

func (s *Service) Store(ctx context.Context, apiKey *domain.APIKey) error {
	if err := apiKey.Validate(); err != nil {
		return err
	}

	apiKey.Key = GenerateSecureToken(16)

	if err := s.repo.Store(ctx, apiKey); err != nil {
		return err
	}

	s.keyCache.Set(apiKey.Key, *apiKey)

	return nil
}

func (s *Service) Update(ctx context.Context, apiKey *domain.APIKey) error {
	if err := apiKey.Validate(); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, apiKey); err != nil {
		return err
	}

	s.keyCache.Del(apiKey.Key)

	return nil
}

func (s *Service) Delete(ctx context.Context, key string) error {
	_, err := s.repo.GetKey(ctx, key)
	if err != nil {
		return err
	}

	err = s.repo.Delete(ctx, key)
	if err != nil {
		return errors.Wrap(err, "could not delete api key: %s", key)
	}

	s.keyCache.Del(key)

	return nil
}

// ValidateAPIKey returns the stored key matching token, or false when no such key exists.
func (s *Service) ValidateAPIKey(ctx context.Context, token string) (*domain.APIKey, bool) {
	if apiKey, ok := s.keyCache.Get(token); ok {
		return &apiKey, true
	}

	found, err := s.repo.GetKey(ctx, token)
	if err != nil {
		s.log.Trace().Msg("invalid api key")
		return nil, false
	}

	s.keyCache.Set(token, *found)

	return found, true
}

func GenerateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
