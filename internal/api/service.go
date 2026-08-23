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
	apiKey.Key = GenerateSecureToken(16)

	if err := s.repo.Store(ctx, apiKey); err != nil {
		return err
	}

	s.keyCache.Set(apiKey.Key, *apiKey)

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

func (s *Service) ValidateAPIKey(ctx context.Context, key string) bool {
	if _, ok := s.keyCache.Get(key); ok {
		return true
	}

	apiKey, err := s.repo.GetKey(ctx, key)
	if err != nil {
		s.log.Trace().Msg("invalid api key")
		return false
	}

	s.keyCache.Set(key, *apiKey)

	return true
}

func GenerateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
