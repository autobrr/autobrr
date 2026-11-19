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

	"github.com/alphadose/haxmap"
	"github.com/rs/zerolog"
)

type repo interface {
	Store(ctx context.Context, key *domain.APIKey) error
	Update(ctx context.Context, key *domain.APIKey) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context) ([]domain.APIKey, error)
	FindByID(ctx context.Context, id int) (*domain.APIKey, error)
	FindByKeyHash(ctx context.Context, keyHash string) (*domain.APIKey, error)
}

type Service struct {
	log  zerolog.Logger
	repo repo

	// m serializes cache fills with key changes. Without it a fill that read a key before an update could cache
	// the old scopes after the update evicted them, and cache hits never go back to the database.
	m        sync.RWMutex
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
	return s.repo.List(ctx)
}

// Store generates a new key and saves its hash. The plain key is set on apiKey for the caller to show once.
func (s *Service) Store(ctx context.Context, apiKey *domain.APIKey) error {
	if err := apiKey.Validate(); err != nil {
		return err
	}

	apiKey.Key = GenerateSecureToken(16)
	apiKey.KeyHash = domain.HashAPIKey(apiKey.Key)

	return s.repo.Store(ctx, apiKey)
}

// Update changes the name and scopes of the key with apiKey.ID.
func (s *Service) Update(ctx context.Context, apiKey *domain.APIKey) error {
	if err := apiKey.Validate(); err != nil {
		return err
	}

	s.m.Lock()
	defer s.m.Unlock()

	if err := s.repo.Update(ctx, apiKey); err != nil {
		return err
	}

	s.keyCache.Del(apiKey.KeyHash)

	return nil
}

func (s *Service) Delete(ctx context.Context, id int) error {
	s.m.Lock()
	defer s.m.Unlock()

	apiKey, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "could not delete api key: %d", id)
	}

	s.keyCache.Del(apiKey.KeyHash)

	return nil
}

// ValidateAPIKey returns the stored key matching token, or false when no such key exists.
func (s *Service) ValidateAPIKey(ctx context.Context, token string) (*domain.APIKey, bool) {
	keyHash := domain.HashAPIKey(token)

	if apiKey, ok := s.keyCache.Get(keyHash); ok {
		return &apiKey, true
	}

	s.m.RLock()
	defer s.m.RUnlock()

	found, err := s.repo.FindByKeyHash(ctx, keyHash)
	if err != nil {
		s.log.Trace().Msg("invalid api key")
		return nil, false
	}

	s.keyCache.Set(keyHash, *found)

	return found, true
}

func GenerateSecureToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
