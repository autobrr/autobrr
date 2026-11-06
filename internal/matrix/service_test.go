// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package matrix

import (
	"context"
	"sync"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memoryRepo struct {
	m        sync.Mutex
	networks map[int64]domain.MatrixNetwork
}

func (r *memoryRepo) ListNetworks(context.Context) ([]domain.MatrixNetwork, error) {
	r.m.Lock()
	defer r.m.Unlock()

	var out []domain.MatrixNetwork
	for _, n := range r.networks {
		out = append(out, n)
	}

	return out, nil
}

func (r *memoryRepo) FindActiveNetworks(ctx context.Context) ([]domain.MatrixNetwork, error) {
	return r.ListNetworks(ctx)
}

func (r *memoryRepo) FindNetworksByHomeserver(_ context.Context, homeserver string) ([]domain.MatrixNetwork, error) {
	r.m.Lock()
	defer r.m.Unlock()

	var out []domain.MatrixNetwork
	for _, n := range r.networks {
		if n.Homeserver == homeserver {
			out = append(out, n)
		}
	}

	return out, nil
}

func (r *memoryRepo) GetNetworkByID(_ context.Context, id int64) (*domain.MatrixNetwork, error) {
	r.m.Lock()
	defer r.m.Unlock()

	n, ok := r.networks[id]
	if !ok {
		return nil, domain.ErrRecordNotFound
	}

	return &n, nil
}

func (r *memoryRepo) StoreNetwork(_ context.Context, network *domain.MatrixNetwork) error {
	r.m.Lock()
	defer r.m.Unlock()

	network.ID = int64(len(r.networks) + 1)
	r.networks[network.ID] = *network

	return nil
}

func (r *memoryRepo) UpdateNetwork(_ context.Context, network *domain.MatrixNetwork) error {
	r.m.Lock()
	defer r.m.Unlock()

	current := r.networks[network.ID]
	if current.Homeserver != network.Homeserver {
		current.AccessToken = ""
	}

	current.Name = network.Name
	current.Enabled = network.Enabled
	current.Homeserver = network.Homeserver
	current.Rooms = network.Rooms
	r.networks[network.ID] = current

	return nil
}

func (r *memoryRepo) DeleteNetwork(_ context.Context, id int64) error {
	r.m.Lock()
	defer r.m.Unlock()

	delete(r.networks, id)

	return nil
}

func (r *memoryRepo) UpdateTokens(_ context.Context, id int64, tokens domain.MatrixTokens) error {
	r.m.Lock()
	defer r.m.Unlock()

	n := r.networks[id]
	n.AuthMethod = tokens.AuthMethod
	n.UserID = tokens.UserID
	n.AccessToken = tokens.AccessToken
	n.RefreshToken = tokens.RefreshToken
	r.networks[id] = n

	return nil
}

func (r *memoryRepo) UpdateRoomID(context.Context, int64, string) error {
	return nil
}

type noIndexers struct{}

func (noIndexers) GetIndexersByMatrixHomeserver(string) []*domain.IndexerDefinition {
	return nil
}

func (noIndexers) IsEnabled(string) bool {
	return false
}

// Tokens verified against one homeserver must never be stored on a network that has
// since moved to another, or the new homeserver would receive them.
func TestService_ReplaceTokensRejectsChangedHomeserver(t *testing.T) {
	repo := &memoryRepo{networks: map[int64]domain.MatrixNetwork{}}
	s := NewService(zerolog.Nop(), &fakeSSE{}, repo, &fakeReleaseService{}, noIndexers{})

	network := domain.MatrixNetwork{Name: "Watchtower", Homeserver: "https://matrix.watchtower.pm"}
	require.NoError(t, repo.StoreNetwork(t.Context(), &network))

	network.Homeserver = "https://matrix.example.org"
	require.NoError(t, s.UpdateNetwork(t.Context(), &network))

	s.m.Lock()
	err := s.replaceTokens(t.Context(), network.ID, "https://matrix.watchtower.pm", domain.MatrixTokens{AuthMethod: domain.MatrixAuthMethodOAuth, AccessToken: "old-server-token"})
	s.m.Unlock()

	require.Error(t, err)

	stored, err := repo.GetNetworkByID(t.Context(), network.ID)
	require.NoError(t, err)
	assert.Empty(t, stored.AccessToken)
}

func TestService_UpdateNetworkCancelsLoginOnHomeserverChange(t *testing.T) {
	repo := &memoryRepo{networks: map[int64]domain.MatrixNetwork{}}
	s := NewService(zerolog.Nop(), &fakeSSE{}, repo, &fakeReleaseService{}, noIndexers{})

	network := domain.MatrixNetwork{Name: "Watchtower", Homeserver: "https://matrix.watchtower.pm"}
	require.NoError(t, repo.StoreNetwork(t.Context(), &network))

	ctx, cancel := context.WithCancel(context.Background())
	s.logins[network.ID] = &deviceLogin{cancel: cancel}

	network.Homeserver = "https://matrix.example.org"
	require.NoError(t, s.UpdateNetwork(t.Context(), &network))

	assert.Error(t, ctx.Err(), "pending login for the old homeserver must be cancelled")
	assert.NotContains(t, s.logins, network.ID)
}
