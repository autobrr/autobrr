// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsMatrixNetwork(t *testing.T) {
	tests := []struct {
		server   string
		expected bool
	}{
		{"matrix.watchtower.pm", true},
		{"matrix://synapse.example.org", true},
		{"matrix.org", true},
		{"chat.matrix.org", true},
		{"subdomain.watchtower.pm", true},
		{"irc.synirc.net", false},
		{"irc.p2p-network.net", false},
		{"chat.freenode.net", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.server, func(t *testing.T) {
			actual := isMatrixNetwork(tc.server)
			assert.Equal(t, tc.expected, actual, "server: %s", tc.server)
		})
	}
}

func TestAccountKey(t *testing.T) {
	assert.Equal(t, "matrix.org/user1", accountKey("matrix.org", "user1"))
	assert.Equal(t, "matrix.org/user1", accountKey("matrix.org", "User1"))
	assert.Equal(t, "matrix.org", accountKey("matrix.org", ""))
}

func TestMatrixCredsPersistence(t *testing.T) {
	tempDir := t.TempDir()
	SetMatrixConfigDir(tempDir)

	server := "test.matrix.server"
	nick := "testuser"

	_, err := loadMatrixCreds(server, nick)
	assert.Error(t, err)

	toSave := &matrixStoredCreds{
		AccessToken:  "mat_token123",
		RefreshToken: "ref_token456",
		ExpiresAt:    time.Now().Add(10 * time.Minute),
		SourcePass:   "secretPass",
		UserID:       "@user:test.matrix",
	}

	err = saveMatrixCreds(server, nick, toSave)
	require.NoError(t, err)

	loaded, err := loadMatrixCreds(server, nick)
	require.NoError(t, err)
	assert.Equal(t, toSave.AccessToken, loaded.AccessToken)
	assert.Equal(t, toSave.RefreshToken, loaded.RefreshToken)
	assert.Equal(t, toSave.SourcePass, loaded.SourcePass)
	assert.Equal(t, toSave.UserID, loaded.UserID)

	err = saveMatrixPendingAuth(server, nick, &matrixDevicePending{
		DeviceCode: "dev_123",
		UserCode:   "ABCD-EFGH",
	})
	require.NoError(t, err)

	err = clearMatrixPendingAuth(server, nick)
	require.NoError(t, err)
}

func TestCheckMatrixTokenActive(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/_matrix/client/v3/account/whoami", r.URL.Path)
		auth := r.Header.Get("Authorization")
		if auth == "Bearer valid_tok" {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"user_id": "@bot:test.matrix"})
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	ctx := context.Background()
	client := ts.Client()

	assert.True(t, checkMatrixTokenActive(ctx, client, ts.URL, "valid_tok"))
	assert.False(t, checkMatrixTokenActive(ctx, client, ts.URL, "invalid_tok"))
}

func TestJoinMatrixRoom(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test_tok", r.Header.Get("Authorization"))

		if strings.Contains(r.URL.Path, "valid-room") {
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"room_id": "!canonicalRoom:test.matrix"})
			return
		}

		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"errcode": "M_NOT_FOUND", "error": "Room not found"})
	}))
	defer ts.Close()

	ctx := context.Background()
	client := ts.Client()

	roomID, err := joinMatrixRoom(ctx, client, ts.URL, "test.matrix", "test_tok", "#valid-room")
	require.NoError(t, err)
	assert.Equal(t, "!canonicalRoom:test.matrix", roomID)

	_, err = joinMatrixRoom(ctx, client, ts.URL, "test.matrix", "test_tok", "#unknown-room")
	require.Error(t, err)
}

func TestRefreshMatrixOAuthToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/oauth/token", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new_access_token",
			"refresh_token": "new_refresh_token",
			"expires_in":    300,
		})
	}))
	defer ts.Close()

	ctx := context.Background()
	client := ts.Client()

	creds := &matrixStoredCreds{
		TokenURI:     ts.URL + "/oauth/token",
		RefreshToken: "old_refresh_token",
	}

	newTok, err := refreshMatrixOAuthToken(ctx, client, creds)
	require.NoError(t, err)
	assert.Equal(t, "new_access_token", newTok)
	assert.Equal(t, "new_refresh_token", creds.RefreshToken)
	assert.False(t, creds.ExpiresAt.IsZero())
}

func TestIsAllowedSenderDomain(t *testing.T) {
	assert.True(t, isAllowedSenderDomain("", "watchtower.pm", "matrix.watchtower.pm"))
	assert.True(t, isAllowedSenderDomain("watchtower.pm", "watchtower.pm", "matrix.watchtower.pm"))
	assert.True(t, isAllowedSenderDomain("WATCHTOWER.PM", "watchtower.pm", "matrix.watchtower.pm"))
	assert.False(t, isAllowedSenderDomain("matrix.org", "watchtower.pm", "matrix.watchtower.pm"))

	assert.True(t, isAllowedSenderDomain("example.org", "", "example.org"))
	assert.True(t, isAllowedSenderDomain("example.org", "", "matrix.example.org"))
	assert.False(t, isAllowedSenderDomain("other.org", "", "matrix.example.org"))
}
