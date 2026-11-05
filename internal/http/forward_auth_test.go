// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/domain"

	"github.com/alexedwards/scs/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func forwardConfig() *domain.Config {
	return &domain.Config{
		BaseURL:               "/",
		BaseURLModeLegacy:     true,
		AuthTrustedUserHeader: "Remote-User",
		AuthTrustedUser:       "ze0s",
		AuthAllowedPeerCIDRs:  []string{"127.0.0.1"},
		CorsAllowedOrigins:    "https://autobrr.example.com",
		AuthLogoutURL:         "https://auth.example.com/logout",
	}
}

func TestForwardCaller(t *testing.T) {
	apiKey := &domain.APIKey{Name: "sonarr", Scopes: []string{"filters:write"}}

	tests := []struct {
		name       string
		user       string
		noUser     bool
		values     []string
		token      string
		session    bool
		wantStatus int
		want       *Caller
	}{
		{name: "user", values: []string{"ze0s"}, wantStatus: http.StatusOK, want: &Caller{Method: domain.AuthMethodForward, Username: "ze0s"}},
		{name: "trimmed", values: []string{"  ze0s "}, wantStatus: http.StatusOK, want: &Caller{Method: domain.AuthMethodForward, Username: "ze0s"}},
		{name: "missing header", wantStatus: http.StatusForbidden},
		{name: "empty header", values: []string{"   "}, wantStatus: http.StatusForbidden},
		{name: "last value wins", values: []string{"admin", "ze0s"}, wantStatus: http.StatusOK, want: &Caller{Method: domain.AuthMethodForward, Username: "ze0s"}},
		{name: "empty last value is not skipped", values: []string{"ze0s", ""}, wantStatus: http.StatusForbidden},
		{name: "trusted user", user: "ze0s", values: []string{"ze0s"}, wantStatus: http.StatusOK, want: &Caller{Method: domain.AuthMethodForward, Username: "ze0s"}},
		{name: "case variant of the trusted user", user: "ze0s", values: []string{"ZE0S"}, wantStatus: http.StatusForbidden},
		{name: "other user", user: "ze0s", values: []string{"admin"}, wantStatus: http.StatusForbidden},
		{name: "no trusted user configured", noUser: true, values: []string{"ze0s"}, wantStatus: http.StatusForbidden},
		{name: "trusted user check uses the last value", user: "ze0s", values: []string{"ze0s", "admin"}, wantStatus: http.StatusForbidden},
		{name: "api key without header", token: "valid", wantStatus: http.StatusOK, want: &Caller{Method: domain.AuthMethodAPIKey, APIKey: apiKey}},
		{name: "api key wins over header", token: "valid", values: []string{"ze0s"}, wantStatus: http.StatusOK, want: &Caller{Method: domain.AuthMethodAPIKey, APIKey: apiKey}},
		{name: "invalid api key with header", token: "nope", values: []string{"ze0s"}, wantStatus: http.StatusUnauthorized},
		{name: "session is not read", session: true, wantStatus: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := forwardConfig()
			if tt.user != "" {
				cfg.AuthTrustedUser = tt.user
			}
			if tt.noUser {
				cfg.AuthTrustedUser = ""
			}

			sm := scs.New()
			s := &Server{
				log:            zerolog.Nop(),
				config:         &config.AppConfig{Config: cfg},
				sessionManager: sm,
				apiService:     &apikeyServiceStub{tokens: map[string]*domain.APIKey{"valid": apiKey}},
			}

			req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
			for _, value := range tt.values {
				req.Header.Add("Remote-User", value)
			}
			if tt.token != "" {
				req.Header.Set("X-API-Token", tt.token)
			}

			if tt.session {
				ctx, err := sm.Load(t.Context(), "")
				require.NoError(t, err)
				sm.Put(ctx, "authenticated", true)
				sm.Put(ctx, "auth_method", "password")
				sm.Put(ctx, "username", "ze0s")
				token, _, err := sm.Commit(ctx)
				require.NoError(t, err)
				req.AddCookie(&http.Cookie{Name: sm.Cookie.Name, Value: token})
			}

			var got *Caller
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got, _ = callerFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			rr := httptest.NewRecorder()
			sm.LoadAndSave(s.IsAuthenticated(next)).ServeHTTP(rr, req)

			assert.Equal(t, tt.wantStatus, rr.Code)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestForwardAuthHandlerChain runs requests through the real middleware chain, so
// the peer allowlist and browser guards in front of the header are covered too.
func TestForwardAuthHandlerChain(t *testing.T) {
	s := NewServer(Deps{
		Log:            zerolog.Nop(),
		Config:         &config.AppConfig{Config: forwardConfig()},
		SessionManager: scs.New(),
		ApiService:     &apikeyServiceStub{},
		OIDCService:    oidcServiceStub{},
	})
	handler := s.Handler()

	serve := func(method, target, remoteAddr string, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, nil)
		req.Host = "autobrr.example.com"
		req.RemoteAddr = remoteAddr
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		return rr
	}

	t.Run("validate returns the forwarded user", func(t *testing.T) {
		rr := serve(http.MethodGet, "/api/auth/validate", "127.0.0.1:5000", map[string]string{"Remote-User": "ze0s"})
		require.Equal(t, http.StatusOK, rr.Code)

		var caller Caller
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&caller))
		assert.Equal(t, Caller{Method: domain.AuthMethodForward, Username: "ze0s"}, caller)
	})

	t.Run("header from a peer outside the allowlist", func(t *testing.T) {
		rr := serve(http.MethodGet, "/api/auth/validate", "192.0.2.10:5000", map[string]string{"Remote-User": "ze0s"})
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("forwarded for cannot spoof the peer", func(t *testing.T) {
		rr := serve(http.MethodGet, "/api/auth/validate", "192.0.2.10:5000", map[string]string{"Remote-User": "ze0s", "X-Forwarded-For": "127.0.0.1"})
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("cross-site browser request with the header", func(t *testing.T) {
		rr := serve(http.MethodGet, "/api/auth/validate", "127.0.0.1:5000", map[string]string{"Remote-User": "ze0s", "Sec-Fetch-Site": "cross-site"})
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("password login is unavailable", func(t *testing.T) {
		rr := serve(http.MethodPost, "/api/auth/login", "127.0.0.1:5000", map[string]string{"Remote-User": "ze0s"})
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("logout is unavailable", func(t *testing.T) {
		rr := serve(http.MethodPost, "/api/auth/logout", "127.0.0.1:5000", map[string]string{"Remote-User": "ze0s"})
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("account management is unavailable", func(t *testing.T) {
		rr := serve(http.MethodPatch, "/api/auth/user/ze0s", "127.0.0.1:5000", map[string]string{"Remote-User": "ze0s"})
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("oidc routes are not mounted", func(t *testing.T) {
		rr := serve(http.MethodGet, "/api/auth/oidc/config", "127.0.0.1:5000", map[string]string{"Remote-User": "ze0s"})
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("config reports the mode and logout url", func(t *testing.T) {
		rr := serve(http.MethodGet, "/api/config/", "127.0.0.1:5000", map[string]string{"Remote-User": "ze0s"})
		require.Equal(t, http.StatusOK, rr.Code)

		var conf configJson
		require.NoError(t, json.NewDecoder(rr.Body).Decode(&conf))
		assert.Equal(t, domain.AuthModeForward, conf.AuthMode)
		assert.Equal(t, domain.AuthCapabilities{APIKeyManagement: true, ProxyGuards: true}, conf.AuthCapabilities)
		assert.Equal(t, "https://auth.example.com/logout", conf.AuthLogoutURL)
	})
}

func TestConfigHidesLogoutURLOutsideForwardMode(t *testing.T) {
	cfg := &domain.Config{AuthLogoutURL: "javascript:alert(1)"}

	router := newConfigHandler(encoder{}, buildInfo{}, &config.AppConfig{Config: cfg})
	rr := httptest.NewRecorder()
	asCaller(testUser)(http.HandlerFunc(router.getConfig)).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, rr.Code)

	var conf configJson
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&conf))
	assert.Empty(t, conf.AuthLogoutURL)
}

func TestAPIKeyHandlerStoreLogsCreatingUser(t *testing.T) {
	var buf bytes.Buffer
	h := newAPIKeyHandler(encoder{}, zerolog.New(&buf), &apikeyServiceStub{}, forwardConfig())

	handler := asCaller(&Caller{Method: domain.AuthMethodForward, Username: "ze0s"})(http.HandlerFunc(h.store))
	rr := serveWithToken(handler, http.MethodPost, "/", "", `{"name":"sonarr","scopes":["filters:write"]}`)

	require.Equal(t, http.StatusCreated, rr.Code)
	assert.Contains(t, buf.String(), `"username":"ze0s"`)
	assert.Contains(t, buf.String(), `"api_key_name":"sonarr"`)
	assert.NotContains(t, buf.String(), "generated-key")
}
