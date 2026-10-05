// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

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

	m        sync.RWMutex
	keyCache map[string]domain.APIKey
}

func NewService(log zerolog.Logger, repo repo) *Service {
	return &Service{
		log:      log.With().Str("module", "api").Logger(),
		repo:     repo,
		keyCache: map[string]domain.APIKey{},
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

	return s.repo.Store(ctx, apiKey)
}

func (s *Service) Update(ctx context.Context, apiKey *domain.APIKey) error {
	if err := apiKey.Validate(); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, apiKey); err != nil {
		return err
	}

	s.m.Lock()
	delete(s.keyCache, apiKey.Key)
	s.m.Unlock()

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

	s.m.Lock()
	delete(s.keyCache, key)
	s.m.Unlock()

	return nil
}

// ValidateAPIKey returns the stored key matching token, or false when no such key exists.
func (s *Service) ValidateAPIKey(ctx context.Context, token string) (*domain.APIKey, bool) {
	s.m.RLock()
	apiKey, ok := s.keyCache[token]
	s.m.RUnlock()

	if ok {
		s.log.Trace().Str("api_key", token).Msg("cache hit")
		return &apiKey, true
	}

	found, err := s.repo.GetKey(ctx, token)
	if err != nil {
		s.log.Trace().Str("api_key", token).Msg("cache invalid key")
		return nil, false
	}

	s.log.Trace().Str("api_key", token).Msg("cache miss")

	s.m.Lock()
	s.keyCache[token] = *found
	s.m.Unlock()

	return found, true
}

func GenerateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
