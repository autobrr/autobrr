// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package matrix

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceLoginFlow(t *testing.T) {
	polls := 0

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/_matrix/client/v1/auth_metadata":
			_ = json.NewEncoder(w).Encode(map[string]string{
				"issuer":                        server.URL + "/",
				"token_endpoint":                server.URL + "/oauth2/token",
				"registration_endpoint":         server.URL + "/oauth2/registration",
				"device_authorization_endpoint": server.URL + "/oauth2/device",
			})

		case "/oauth2/registration":
			var metadata map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&metadata))
			assert.Equal(t, "none", metadata["token_endpoint_auth_method"])
			assert.Equal(t, []any{}, metadata["response_types"])
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"client_id":"client-1"}`))

		case "/oauth2/device":
			require.NoError(t, r.ParseForm())
			assert.Equal(t, "client-1", r.Form.Get("client_id"))
			assert.Equal(t, "urn:matrix:client:api:* urn:matrix:client:device:DEVICE1234", r.Form.Get("scope"))
			_, _ = w.Write([]byte(`{"device_code":"dc","user_code":"ABCD-EFGH","verification_uri":"https://auth.example/link","expires_in":600}`))

		case "/oauth2/token":
			require.NoError(t, r.ParseForm())
			assert.Equal(t, grantTypeDeviceCode, r.Form.Get("grant_type"))

			polls++
			if polls == 1 {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
				return
			}

			_, _ = w.Write([]byte(`{"access_token":"at","refresh_token":"rt","expires_in":300,"token_type":"Bearer"}`))

		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	ctx := t.Context()

	metadata, err := NewClient(server.URL, server.Client()).AuthMetadata(ctx)
	require.NoError(t, err)

	oauth := NewOAuthClient(server.Client())

	clientID, err := oauth.RegisterClient(ctx, metadata, DefaultClientMetadata())
	require.NoError(t, err)
	assert.Equal(t, "client-1", clientID)

	authorization, err := oauth.RequestDeviceAuthorization(ctx, metadata, clientID, "DEVICE1234")
	require.NoError(t, err)
	assert.Equal(t, "ABCD-EFGH", authorization.UserCode)
	assert.Equal(t, 5, authorization.Interval, "missing interval defaults to the RFC 8628 minimum")

	_, err = oauth.PollDeviceToken(ctx, metadata, clientID, authorization.DeviceCode)
	oerr, ok := errors.AsType[*OAuthError](err)
	require.True(t, ok)
	assert.Equal(t, OAuthErrAuthorizationPending, oerr.Code)

	token, err := oauth.PollDeviceToken(ctx, metadata, clientID, authorization.DeviceCode)
	require.NoError(t, err)
	assert.Equal(t, "at", token.AccessToken)
	assert.Equal(t, "rt", token.RefreshToken)

	now := time.Now()
	assert.Equal(t, now.Add(300*time.Second), token.ExpiresAt(now))
}

func TestRefreshTokenKeepsOldRefreshTokenWhenNotRotated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"access_token":"new","expires_in":300}`))
	}))
	defer server.Close()

	token, err := NewOAuthClient(server.Client()).RefreshToken(t.Context(), server.URL, "client", "old-refresh")
	require.NoError(t, err)
	assert.Equal(t, "new", token.AccessToken)
	assert.Equal(t, "old-refresh", token.RefreshToken)
}

func TestErrorDecoding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"errcode":"M_LIMIT_EXCEEDED","error":"slow down","retry_after_ms":1500}`))
	}))
	defer server.Close()

	_, err := NewClient(server.URL, server.Client()).Whoami(t.Context())

	merr, ok := errors.AsType[*Error](err)
	require.True(t, ok)
	assert.Equal(t, "M_LIMIT_EXCEEDED", merr.ErrCode)
	assert.Equal(t, 1500*time.Millisecond, merr.RetryAfter)
	assert.False(t, merr.IsUnauthorized())
}

func TestNewDeviceID(t *testing.T) {
	id := NewDeviceID()
	assert.Len(t, id, 10)
	assert.Empty(t, strings.Trim(id, deviceIDAlphabet))
}
