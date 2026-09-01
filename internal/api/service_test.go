// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package api

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type apiRepoMock struct {
	m           sync.Mutex
	keys        map[string]domain.APIKey
	getKeyCalls int
}

func newAPIRepoMock(keys ...domain.APIKey) *apiRepoMock {
	r := &apiRepoMock{keys: map[string]domain.APIKey{}}
	for _, key := range keys {
		r.keys[key.Key] = key
	}
	return r
}

func (r *apiRepoMock) Store(_ context.Context, key *domain.APIKey) error {
	r.m.Lock()
	defer r.m.Unlock()

	r.keys[key.Key] = *key

	return nil
}

func (r *apiRepoMock) Delete(_ context.Context, key string) error {
	r.m.Lock()
	defer r.m.Unlock()

	delete(r.keys, key)

	return nil
}

func (r *apiRepoMock) GetAllAPIKeys(_ context.Context) ([]domain.APIKey, error) {
	r.m.Lock()
	defer r.m.Unlock()

	keys := make([]domain.APIKey, 0, len(r.keys))
	for _, key := range r.keys {
		keys = append(keys, key)
	}

	return keys, nil
}

func (r *apiRepoMock) GetKey(_ context.Context, key string) (*domain.APIKey, error) {
	r.m.Lock()
	defer r.m.Unlock()

	r.getKeyCalls++

	k, ok := r.keys[key]
	if !ok {
		return nil, errors.New("api key not found")
	}

	return &k, nil
}

func TestService_ValidateAPIKey(t *testing.T) {
	repo := newAPIRepoMock(domain.APIKey{Name: "test", Key: "valid-key"})
	s := NewService(zerolog.Nop(), repo)

	assert.True(t, s.ValidateAPIKey(context.Background(), "valid-key"))
	assert.False(t, s.ValidateAPIKey(context.Background(), "invalid-key"))

	// second validation must come from the cache, not the repo
	assert.True(t, s.ValidateAPIKey(context.Background(), "valid-key"))
	assert.Equal(t, 2, repo.getKeyCalls)
}

func TestService_ListReturnsAllKeysAfterValidation(t *testing.T) {
	// a warmed cache must not shadow the repo: List previously returned only the
	// validated subset once any key had been cached
	repo := newAPIRepoMock(
		domain.APIKey{Name: "one", Key: "key-one"},
		domain.APIKey{Name: "two", Key: "key-two"},
	)
	s := NewService(zerolog.Nop(), repo)

	assert.True(t, s.ValidateAPIKey(context.Background(), "key-one"))

	keys, err := s.List(context.Background())
	assert.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestService_DeleteEvictsCache(t *testing.T) {
	repo := newAPIRepoMock()
	s := NewService(zerolog.Nop(), repo)

	apiKey := &domain.APIKey{Name: "test"}
	assert.NoError(t, s.Store(context.Background(), apiKey))
	assert.True(t, s.ValidateAPIKey(context.Background(), apiKey.Key))

	assert.NoError(t, s.Delete(context.Background(), apiKey.Key))
	assert.False(t, s.ValidateAPIKey(context.Background(), apiKey.Key))
}

func TestService_ValidateAPIKeyConcurrent(t *testing.T) {
	// exercises parallel cache reads and writes; meaningful under -race, where
	// the previous plain map was a process-fatal data race
	keys := make([]domain.APIKey, 0, 10)
	for i := range 10 {
		keys = append(keys, domain.APIKey{Name: fmt.Sprintf("key-%d", i), Key: fmt.Sprintf("valid-%d", i)})
	}

	s := NewService(zerolog.Nop(), newAPIRepoMock(keys...))

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			assert.True(t, s.ValidateAPIKey(context.Background(), fmt.Sprintf("valid-%d", i%10)))
			assert.False(t, s.ValidateAPIKey(context.Background(), fmt.Sprintf("invalid-%d", i)))
		}()
	}
	wg.Wait()
}
