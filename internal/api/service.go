// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

type Service interface {
	List(ctx context.Context) ([]domain.APIKey, error)
	Store(ctx context.Context, key *domain.APIKey) error
	Delete(ctx context.Context, key string) error
	ValidateAPIKey(ctx context.Context, token string) bool
}

type service struct {
	log  zerolog.Logger
	repo domain.APIRepo

	keyCache map[string]domain.APIKey
}

func NewService(log logger.Logger, repo domain.APIRepo) Service {
	return &service{
		log:      log.With().Str("module", "api").Logger(),
		repo:     repo,
		keyCache: map[string]domain.APIKey{},
	}
}

func (s *service) List(ctx context.Context) ([]domain.APIKey, error) {
	if len(s.keyCache) > 0 {
		keys := make([]domain.APIKey, 0, len(s.keyCache))

		for _, key := range s.keyCache {
			keys = append(keys, key)
		}

		return keys, nil
	}

	return s.repo.GetAllAPIKeys(ctx)
}

func (s *service) Store(ctx context.Context, apiKey *domain.APIKey) error {
	apiKey.Key = GenerateSecureToken(16)

	if err := s.repo.Store(ctx, apiKey); err != nil {
		return err
	}

	if len(s.keyCache) > 0 {
		// set new apiKey
		s.keyCache[apiKey.Key] = *apiKey
	}

	return nil
}

func (s *service) Delete(ctx context.Context, key string) error {
	_, err := s.repo.GetKey(ctx, key)
	if err != nil {
		return err
	}

	err = s.repo.Delete(ctx, key)
	if err != nil {
		return errors.Wrap(err, "could not delete api key: %s", key)
	}

	// remove key from cache
	delete(s.keyCache, key)

	return nil
}

func (s *service) ValidateAPIKey(ctx context.Context, key string) bool {
	if _, ok := s.keyCache[key]; ok {
		s.log.Trace().Msgf("api service key cache hit: %s", key)
		return true
	}

	apiKey, err := s.repo.GetKey(ctx, key)
	if err != nil {
		s.log.Trace().Msgf("api service key cache invalid key: %s", key)
		return false
	}

	s.log.Trace().Msgf("api service key cache miss: %s", key)

	s.keyCache[key] = *apiKey

	return true
}

func GenerateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
