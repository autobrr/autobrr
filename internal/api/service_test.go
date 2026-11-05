// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package api

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type apiRepoMock struct {
	m           sync.Mutex
	keys        map[int]domain.APIKey
	nextID      int
	getKeyCalls int

	// beforeFindReturn runs once inside FindByKeyHash, after the row was read.
	beforeFindReturn func()
}

func newAPIRepoMock(tokens ...string) *apiRepoMock {
	r := &apiRepoMock{keys: map[int]domain.APIKey{}}
	for _, token := range tokens {
		r.nextID++
		r.keys[r.nextID] = domain.APIKey{ID: r.nextID, Name: token, KeyHash: domain.HashAPIKey(token), Scopes: []string{"filters:write"}}
	}
	return r
}

func (r *apiRepoMock) Store(_ context.Context, key *domain.APIKey) error {
	r.m.Lock()
	defer r.m.Unlock()

	r.nextID++
	key.ID = r.nextID

	stored := *key
	stored.Key = ""
	r.keys[key.ID] = stored

	return nil
}

func (r *apiRepoMock) Update(_ context.Context, key *domain.APIKey) error {
	r.m.Lock()
	defer r.m.Unlock()

	existing, ok := r.keys[key.ID]
	if !ok {
		return domain.ErrRecordNotFound
	}

	key.KeyHash = existing.KeyHash
	r.keys[key.ID] = *key

	return nil
}

func (r *apiRepoMock) Delete(_ context.Context, id int) error {
	r.m.Lock()
	defer r.m.Unlock()

	delete(r.keys, id)

	return nil
}

func (r *apiRepoMock) List(_ context.Context) ([]domain.APIKey, error) {
	r.m.Lock()
	defer r.m.Unlock()

	keys := make([]domain.APIKey, 0, len(r.keys))
	for _, key := range r.keys {
		keys = append(keys, key)
	}

	return keys, nil
}

func (r *apiRepoMock) FindByID(_ context.Context, id int) (*domain.APIKey, error) {
	r.m.Lock()
	defer r.m.Unlock()

	k, ok := r.keys[id]
	if !ok {
		return nil, domain.ErrRecordNotFound
	}

	return &k, nil
}

func (r *apiRepoMock) FindByKeyHash(_ context.Context, keyHash string) (*domain.APIKey, error) {
	r.m.Lock()

	r.getKeyCalls++

	var found *domain.APIKey
	for _, k := range r.keys {
		if k.KeyHash == keyHash {
			found = &k
			break
		}
	}

	hook := r.beforeFindReturn
	r.beforeFindReturn = nil

	r.m.Unlock()

	if hook != nil {
		hook()
	}

	if found == nil {
		return nil, domain.ErrRecordNotFound
	}

	return found, nil
}

func TestService_ValidateAPIKey(t *testing.T) {
	repo := newAPIRepoMock("valid-key")
	s := NewService(zerolog.Nop(), repo)

	apiKey, ok := s.ValidateAPIKey(context.Background(), "valid-key")
	assert.True(t, ok)
	assert.Equal(t, 1, apiKey.ID)
	assert.Equal(t, []string{"filters:write"}, apiKey.Scopes)
	_, ok = s.ValidateAPIKey(context.Background(), "invalid-key")
	assert.False(t, ok)

	// second validation must come from the cache, not the repo
	apiKey, ok = s.ValidateAPIKey(context.Background(), "valid-key")
	assert.True(t, ok)
	assert.Equal(t, 1, apiKey.ID)
	assert.Equal(t, []string{"filters:write"}, apiKey.Scopes)
	assert.Equal(t, 2, repo.getKeyCalls)
}

func TestService_StoreKeepsOnlyHash(t *testing.T) {
	repo := newAPIRepoMock()
	s := NewService(zerolog.Nop(), repo)

	apiKey := &domain.APIKey{Name: "test", Scopes: []string{"filters:write"}}
	require.NoError(t, s.Store(context.Background(), apiKey))

	assert.NotEmpty(t, apiKey.Key)
	assert.Equal(t, domain.HashAPIKey(apiKey.Key), apiKey.KeyHash)

	keys, err := s.List(context.Background())
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Empty(t, keys[0].Key)
}

func TestService_ListReturnsAllKeysAfterValidation(t *testing.T) {
	// a warmed cache must not shadow the repo: List previously returned only the
	// validated subset once any key had been cached
	repo := newAPIRepoMock("key-one", "key-two")
	s := NewService(zerolog.Nop(), repo)

	_, ok := s.ValidateAPIKey(context.Background(), "key-one")
	assert.True(t, ok)

	keys, err := s.List(context.Background())
	assert.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestService_DeleteEvictsCache(t *testing.T) {
	repo := newAPIRepoMock()
	s := NewService(zerolog.Nop(), repo)

	apiKey := &domain.APIKey{Name: "test", Scopes: []string{"filters:write"}}
	assert.NoError(t, s.Store(context.Background(), apiKey))
	_, ok := s.ValidateAPIKey(context.Background(), apiKey.Key)
	assert.True(t, ok)

	assert.NoError(t, s.Delete(context.Background(), apiKey.ID))
	_, ok = s.ValidateAPIKey(context.Background(), apiKey.Key)
	assert.False(t, ok)
}

func TestService_UpdateEvictsCache(t *testing.T) {
	repo := newAPIRepoMock("valid-key")
	s := NewService(zerolog.Nop(), repo)

	_, ok := s.ValidateAPIKey(context.Background(), "valid-key")
	require.True(t, ok)

	require.NoError(t, s.Update(context.Background(), &domain.APIKey{ID: 1, Name: "valid-key", Scopes: []string{"filters:read"}}))

	apiKey, ok := s.ValidateAPIKey(context.Background(), "valid-key")
	require.True(t, ok)
	assert.Equal(t, []string{"filters:read"}, apiKey.Scopes)
}

func TestService_UpdateDuringCacheFill(t *testing.T) {
	// a cache fill that read the key before an update must not cache the old scopes after the update
	repo := newAPIRepoMock("valid-key")
	s := NewService(zerolog.Nop(), repo)

	fillRead := make(chan struct{})
	releaseFill := make(chan struct{})
	repo.beforeFindReturn = func() {
		close(fillRead)
		<-releaseFill
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		s.ValidateAPIKey(context.Background(), "valid-key")
	})

	<-fillRead

	updateDone := make(chan error, 1)
	go func() {
		updateDone <- s.Update(context.Background(), &domain.APIKey{ID: 1, Name: "valid-key", Scopes: []string{"filters:read"}})
	}()

	// give an unsynchronized Update the chance to finish before the fill writes the cache
	select {
	case err := <-updateDone:
		require.NoError(t, err)
		updateDone <- nil
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseFill)
	wg.Wait()
	require.NoError(t, <-updateDone)

	apiKey, ok := s.ValidateAPIKey(context.Background(), "valid-key")
	require.True(t, ok)
	assert.Equal(t, []string{"filters:read"}, apiKey.Scopes)
}

func TestService_ValidateAPIKeyConcurrent(t *testing.T) {
	// exercises parallel cache reads and writes; meaningful under -race, where
	// the previous plain map was a process-fatal data race
	tokens := make([]string, 0, 10)
	for i := range 10 {
		tokens = append(tokens, fmt.Sprintf("valid-%d", i))
	}

	s := NewService(zerolog.Nop(), newAPIRepoMock(tokens...))

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() {
			_, ok := s.ValidateAPIKey(context.Background(), fmt.Sprintf("valid-%d", i%10))
			assert.True(t, ok)
			_, ok = s.ValidateAPIKey(context.Background(), fmt.Sprintf("invalid-%d", i))
			assert.False(t, ok)
		})
	}
	wg.Wait()
}
