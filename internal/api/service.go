// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/rs/zerolog"
)

type Service interface {
	List(ctx context.Context) ([]domain.APIKey, error)
	Store(ctx context.Context, key *domain.APIKey) error
	Update(ctx context.Context, key *domain.APIKey) error
	Delete(ctx context.Context, key string) error
	ValidateAPIKey(ctx context.Context, token string) bool
}

type service struct {
	log  zerolog.Logger
	repo domain.APIRepo

	keyCache []domain.APIKey
}

func NewService(log logger.Logger, repo domain.APIRepo) Service {
	return &service{
		log:      log.With().Str("module", "api").Logger(),
		repo:     repo,
		keyCache: []domain.APIKey{},
	}
}

func (s *service) List(ctx context.Context) ([]domain.APIKey, error) {
	if len(s.keyCache) > 0 {
		return s.keyCache, nil
	}

	return s.repo.GetKeys(ctx)
}

func (s *service) Store(ctx context.Context, key *domain.APIKey) error {

	key.Key = GenerateSecureToken(key.KeyLength)

	if err := s.repo.Store(ctx, key); err != nil {
		return err
	}

	if len(s.keyCache) > 0 {
		// set new key
		s.keyCache = append(s.keyCache, *key)
	}

	return nil
}

func (s *service) Update(ctx context.Context, key *domain.APIKey) error {
	return nil
}

func (s *service) Delete(ctx context.Context, key string) error {
	// reset
	s.keyCache = []domain.APIKey{}

	return s.repo.Delete(ctx, key)
}

func (s *service) ValidateAPIKey(ctx context.Context, key string) bool {
	keys, err := s.repo.GetKeys(ctx)
	if err != nil {
		return false
	}

	for _, k := range keys {
		if k.Key == key {
			return true
		}
	}
	return false
}

func GenerateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
