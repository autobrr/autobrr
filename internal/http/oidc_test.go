// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package http

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/alexedwards/scs/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type oidcConfigServiceMock struct {
	oidcAuthServiceMock
	pkce bool
}

func (s *oidcConfigServiceMock) SupportsPKCE() bool {
	return s.pkce
}

func (s *oidcConfigServiceMock) OauthAuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	cfg := oauth2.Config{
		ClientID:    "autobrr",
		RedirectURL: "http://localhost:7474/api/auth/oidc/callback",
		Endpoint:    oauth2.Endpoint{AuthURL: "https://auth.example.com/authorize"},
	}
	return cfg.AuthCodeURL(state, opts...)
}

func TestOIDCConfigAutoRedirect(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name          string
		enabled       bool
		autoRedirect  bool
		authenticated bool
		pkce          bool
		wantRedirect  bool
	}{
		{name: "disabled by default", enabled: true},
		{name: "enabled", enabled: true, autoRedirect: true, wantRedirect: true},
		{name: "enabled with PKCE", enabled: true, autoRedirect: true, pkce: true, wantRedirect: true},
		{name: "authenticated session", enabled: true, autoRedirect: true, authenticated: true, pkce: true},
		{name: "OIDC disabled", autoRedirect: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sessions := scs.New()
			handler := NewOIDCHandler(encoder{}, zerolog.Nop(), &domain.Config{
				OIDCEnabled:      tt.enabled,
				OIDCAutoRedirect: tt.autoRedirect,
			}, sessions, &oidcConfigServiceMock{pkce: tt.pkce}, func(*http.Request) string { return "" })

			var storedState, storedVerifier string
			next := sessions.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sessions.Put(r.Context(), "authenticated", tt.authenticated)
				sessions.Put(r.Context(), "oidc_state", "previous-state")
				sessions.Put(r.Context(), "oidc_pkce_verifier", "previous-verifier")
				handler.getConfig(w, r)
				storedState = sessions.GetString(r.Context(), "oidc_state")
				storedVerifier = sessions.GetString(r.Context(), "oidc_pkce_verifier")
			}))
			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/oidc/config", nil))
			require.Equal(t, http.StatusOK, rec.Code)

			var response GetConfigResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, tt.wantRedirect, response.AutoRedirect)
			authURL, err := url.Parse(response.AuthorizationURL)
			require.NoError(t, err)
			require.NotEmpty(t, response.State)
			assert.Equal(t, response.State, authURL.Query().Get("state"))

			if tt.authenticated {
				assert.Equal(t, "previous-state", storedState)
				assert.Equal(t, "previous-verifier", storedVerifier)
				return
			}

			assert.Equal(t, response.State, storedState)
			if tt.pkce {
				require.NotEmpty(t, storedVerifier)
				challenge := sha256.Sum256([]byte(storedVerifier))
				assert.Equal(t, base64.RawURLEncoding.EncodeToString(challenge[:]), authURL.Query().Get("code_challenge"))
				assert.Equal(t, "S256", authURL.Query().Get("code_challenge_method"))
			} else {
				assert.Empty(t, storedVerifier)
				assert.Empty(t, authURL.Query().Get("code_challenge"))
			}
		})
	}
}

func TestOIDCConfigStoresRedirect(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		baseURL  string
		redirect string
		want     string
	}{
		{name: "page under root", baseURL: "/", redirect: "/filters/12?tab=actions", want: "/filters/12?tab=actions"},
		{name: "page under base URL", baseURL: "/autobrr/", redirect: "/autobrr/filters/12", want: "/autobrr/filters/12"},
		{name: "outside base URL", baseURL: "/autobrr/", redirect: "/filters/12"},
		{name: "other origin", baseURL: "/", redirect: "https://evil.example/"},
		{name: "no redirect", baseURL: "/"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sessions := scs.New()
			handler := NewOIDCHandler(encoder{}, zerolog.Nop(), &domain.Config{OIDCEnabled: true, BaseURL: tt.baseURL}, sessions, &oidcConfigServiceMock{}, func(*http.Request) string { return "" })

			var stored string
			next := sessions.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sessions.Put(r.Context(), "oidc_redirect", "/stale")
				handler.getConfig(w, r)
				stored = sessions.GetString(r.Context(), "oidc_redirect")
			}))
			target := "/api/auth/oidc/config"
			if tt.redirect != "" {
				target += "?redirect=" + url.QueryEscape(tt.redirect)
			}
			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.want, stored)
		})
	}
}

func TestIsRedirectUnderBase(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		target  string
		baseURL string
		want    bool
	}{
		{target: "/", baseURL: "/", want: true},
		{target: "/filters/12?tab=actions#top", baseURL: "/", want: true},
		{target: "/autobrr/", baseURL: "/autobrr/", want: true},
		{target: "/autobrr/settings", baseURL: "/autobrr", want: true},
		{target: "/autobrrx/settings", baseURL: "/autobrr"},
		{target: "/settings", baseURL: "/autobrr/"},
		{target: "/autobrr/../evil", baseURL: "/autobrr/"},
		{target: "/autobrr/%2e%2e/evil", baseURL: "/autobrr/"},
		{target: "//evil.example", baseURL: "/"},
		{target: "/\\evil.example", baseURL: "/"},
		{target: "/\t/evil.example", baseURL: "/"},
		{target: "https://evil.example/", baseURL: "/"},
		{target: "javascript:alert(1)", baseURL: "/"},
		{target: "filters", baseURL: "/"},
	} {
		t.Run(tt.target, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isRedirectUnderBase(tt.target, tt.baseURL))
		})
	}
}
