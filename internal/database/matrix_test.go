// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getMockMatrixNetwork() domain.MatrixNetwork {
	return domain.MatrixNetwork{
		Name:       "Watchtower",
		Enabled:    true,
		Homeserver: "https://matrix.watchtower.pm",
		Rooms: []domain.MatrixRoom{
			{Enabled: true, Name: "#blu-announce:matrix.watchtower.pm"},
		},
	}
}

func TestMatrixRepo_NetworkLifecycle(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		repo := NewMatrixRepo(setupLoggerForTest(), testDb.db)

		t.Run(fmt.Sprintf("StoreUpdateDelete [%s]", dbType), func(t *testing.T) {
			network := getMockMatrixNetwork()
			require.NoError(t, repo.StoreNetwork(ctx, &network))
			require.NotZero(t, network.ID)

			defer func() {
				_ = repo.DeleteNetwork(ctx, network.ID)
			}()

			stored, err := repo.GetNetworkByID(ctx, network.ID)
			require.NoError(t, err)
			assert.Equal(t, network.Homeserver, stored.Homeserver)
			require.Len(t, stored.Rooms, 1)
			assert.Empty(t, stored.Rooms[0].RoomID)

			require.NoError(t, repo.UpdateRoomID(ctx, stored.Rooms[0].ID, "!room:matrix.watchtower.pm"))

			expiresAt := time.Now().Add(5 * time.Minute).Truncate(time.Second)
			tokens := domain.MatrixTokens{
				AuthMethod:         domain.MatrixAuthMethodOAuth,
				UserID:             "@autobrr:matrix.watchtower.pm",
				DeviceID:           "DEVICE",
				AccessToken:        "access",
				RefreshToken:       "refresh",
				TokenExpiresAt:     &expiresAt,
				OAuthClientID:      "client",
				OAuthTokenEndpoint: "https://auth.watchtower.pm/oauth2/token",
			}
			require.NoError(t, repo.UpdateTokens(ctx, network.ID, tokens))

			// a settings update keeps tokens and resolved room ids and adds new rooms
			stored.Name = "Watchtower renamed"
			stored.Rooms = append(stored.Rooms, domain.MatrixRoom{Enabled: true, Name: "#other:matrix.watchtower.pm"})
			require.NoError(t, repo.UpdateNetwork(ctx, stored))

			updated, err := repo.GetNetworkByID(ctx, network.ID)
			require.NoError(t, err)
			assert.Equal(t, "Watchtower renamed", updated.Name)
			assert.Equal(t, "refresh", updated.RefreshToken)
			assert.True(t, updated.CanRefresh())
			require.NotNil(t, updated.TokenExpiresAt)
			assert.True(t, expiresAt.Equal(*updated.TokenExpiresAt))
			require.Len(t, updated.Rooms, 2)
			assert.Equal(t, "!room:matrix.watchtower.pm", updated.Rooms[0].RoomID)

			// dropping a room deletes it
			updated.Rooms = updated.Rooms[1:]
			require.NoError(t, repo.UpdateNetwork(ctx, updated))

			byHomeserver, err := repo.FindNetworksByHomeserver(ctx, network.Homeserver)
			require.NoError(t, err)
			require.Len(t, byHomeserver, 1)
			require.Len(t, byHomeserver[0].Rooms, 1)
			assert.Equal(t, "#other:matrix.watchtower.pm", byHomeserver[0].Rooms[0].Name)

			// a new homeserver drops the credentials of the old one
			require.NoError(t, repo.UpdateTokens(ctx, network.ID, tokens))
			byHomeserver[0].Homeserver = "https://matrix.example.org"
			require.NoError(t, repo.UpdateNetwork(ctx, &byHomeserver[0]))

			moved, err := repo.GetNetworkByID(ctx, network.ID)
			require.NoError(t, err)
			assert.False(t, moved.HasCredentials())
			assert.Empty(t, moved.RefreshToken)
			assert.Empty(t, moved.OAuthClientID)

			require.NoError(t, repo.UpdateTokens(ctx, network.ID, domain.MatrixTokens{}))

			cleared, err := repo.GetNetworkByID(ctx, network.ID)
			require.NoError(t, err)
			assert.False(t, cleared.HasCredentials())
			assert.Nil(t, cleared.TokenExpiresAt)

			require.NoError(t, repo.DeleteNetwork(ctx, network.ID))

			_, err = repo.GetNetworkByID(ctx, network.ID)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)

			active, err := repo.FindActiveNetworks(ctx)
			require.NoError(t, err)
			assert.Empty(t, active)
		})
	}
}
