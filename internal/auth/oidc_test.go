// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func newTestOIDCService(issuer string) *OIDCService {
	return NewOIDCService(zerolog.Nop(), &domain.Config{
		OIDCEnabled:      true,
		OIDCIssuer:       issuer,
		OIDCClientID:     "autobrr",
		OIDCClientSecret: "secret",
		OIDCRedirectURL:  "http://localhost:7474/api/auth/oidc/callback",
	})
}

// newTestProvider serves a discovery document once up is true.
func newTestProvider(t *testing.T, up *atomic.Bool) *httptest.Server {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !up.Load() || r.URL.Path != "/.well-known/openid-configuration" {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                           srv.URL,
			"authorization_endpoint":           srv.URL + "/authorize",
			"token_endpoint":                   srv.URL + "/token",
			"jwks_uri":                         srv.URL + "/jwks",
			"code_challenge_methods_supported": []string{"S256"},
		})
	}))
	t.Cleanup(srv.Close)

	return srv
}

func TestOIDCServiceUnavailableProvider(t *testing.T) {
	t.Parallel()
	var up atomic.Bool
	srv := newTestProvider(t, &up)
	s := newTestOIDCService(srv.URL)

	require.NoError(t, s.Discover(t.Context()))

	assert.False(t, s.IsReady())
	assert.False(t, s.SupportsPKCE())
	assert.Empty(t, s.OauthAuthCodeURL("state"))
	assert.Empty(t, s.GetIssuer())
	assert.Equal(t, oauth2.Endpoint{}, s.GetEndpoint())

	_, err := s.OAuthExchange(t.Context(), "code")
	assert.ErrorIs(t, err, ErrProviderUnavailable)
	_, err = s.VerifyIDToken(t.Context(), "token")
	assert.ErrorIs(t, err, ErrProviderUnavailable)
	_, err = s.UserInfo(t.Context(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}))
	assert.ErrorIs(t, err, ErrProviderUnavailable)
}

func TestOIDCServiceDiscoversProvider(t *testing.T) {
	t.Parallel()
	var up atomic.Bool
	up.Store(true)
	srv := newTestProvider(t, &up)
	s := newTestOIDCService(srv.URL)

	require.NoError(t, s.Discover(t.Context()))

	require.Eventually(t, s.IsReady, 5*time.Second, 10*time.Millisecond)
	assert.Equal(t, srv.URL, s.GetIssuer())
	assert.True(t, s.SupportsPKCE())
	assert.Contains(t, s.OauthAuthCodeURL("state"), srv.URL+"/authorize?")
}

func TestOIDCServiceDiscoveryRetriesUntilProviderResponds(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for the first retry delay")
	}
	t.Parallel()
	var up atomic.Bool
	srv := newTestProvider(t, &up)
	s := newTestOIDCService(srv.URL)

	require.NoError(t, s.Discover(t.Context()))
	time.Sleep(100 * time.Millisecond)
	require.False(t, s.IsReady())

	up.Store(true)
	require.Eventually(t, s.IsReady, oidcDiscoveryMinDelay+5*time.Second, 50*time.Millisecond)
}

func TestOIDCServiceValidateConfigRejectsBothAllowedUserSettings(t *testing.T) {
	s := newTestOIDCService("https://auth.example.com")
	require.NoError(t, s.ValidateConfig())

	s.cfg.OIDCAllowedSubject = "abc-123"
	require.NoError(t, s.ValidateConfig())

	s.cfg.OIDCAllowedEmail = "me@example.com"
	assert.ErrorContains(t, s.ValidateConfig(), "set only one of oidcAllowedSubject and oidcAllowedEmail")
}
