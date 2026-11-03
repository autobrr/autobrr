// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package http

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/domain"

	"github.com/alexedwards/scs/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const (
	testOIDCIssuer   = "https://idp.example.com"
	testOIDCClientID = "autobrr"
)

// oidcCallbackServiceMock verifies real RS256 ID tokens against a static key, so
// the callback runs go-oidc's verification without a provider to talk to.
type oidcCallbackServiceMock struct {
	verifier *oidc.IDTokenVerifier
	idToken  string
	userInfo *oidc.UserInfo
}

func (s *oidcCallbackServiceMock) IsEnabled() bool       { return true }
func (s *oidcCallbackServiceMock) ValidateConfig() error { return nil }
func (s *oidcCallbackServiceMock) SupportsPKCE() bool    { return false }

func (s *oidcCallbackServiceMock) GetEndpoint() oauth2.Endpoint { return oauth2.Endpoint{} }

func (s *oidcCallbackServiceMock) OauthAuthCodeURL(string, ...oauth2.AuthCodeOption) string {
	return ""
}

func (s *oidcCallbackServiceMock) OAuthExchange(context.Context, string, ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return (&oauth2.Token{AccessToken: "access"}).WithExtra(map[string]any{"id_token": s.idToken}), nil
}

func (s *oidcCallbackServiceMock) VerifyIDToken(ctx context.Context, idToken string) (*oidc.IDToken, error) {
	return s.verifier.Verify(ctx, idToken)
}

func (s *oidcCallbackServiceMock) UserInfo(context.Context, oauth2.TokenSource) (*oidc.UserInfo, error) {
	return s.userInfo, nil
}

func signTestIDToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()

	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	require.NoError(t, err)

	payload, err := json.Marshal(claims)
	require.NoError(t, err)

	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(signingInput))

	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	require.NoError(t, err)

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

// runOIDCCallback drives a callback with a valid state and returns the response
// along with the resulting session token.
func runOIDCCallback(t *testing.T, cfg *domain.Config, idClaims map[string]any, userInfo *oidc.UserInfo) (*httptest.ResponseRecorder, *scs.SessionManager, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	claims := map[string]any{
		"iss": testOIDCIssuer,
		"aud": testOIDCClientID,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	maps.Copy(claims, idClaims)

	service := &oidcCallbackServiceMock{
		verifier: oidc.NewVerifier(testOIDCIssuer, &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&key.PublicKey}}, &oidc.Config{ClientID: testOIDCClientID}),
		idToken:  signTestIDToken(t, key, claims),
		userInfo: userInfo,
	}

	sm := scs.New()
	handler := NewOIDCHandler(encoder{}, zerolog.Nop(), cfg, sm, service, func(*http.Request) string { return "" })

	ctx, err := sm.Load(t.Context(), "")
	require.NoError(t, err)
	sm.Put(ctx, "oidc_state", "state")
	token, _, err := sm.Commit(ctx)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/callback?state=state&code=code", nil)
	req.AddCookie(&http.Cookie{Name: sm.Cookie.Name, Value: token})
	rec := httptest.NewRecorder()
	sm.LoadAndSave(http.HandlerFunc(handler.handleCallback)).ServeHTTP(rec, req)

	for _, c := range rec.Result().Cookies() {
		if c.Name == sm.Cookie.Name {
			token = c.Value
		}
	}

	return rec, sm, token
}

func TestOIDCCallbackAllowedUser(t *testing.T) {
	tests := []struct {
		name        string
		allowedUser string
		idClaims    map[string]any
		userInfo    *oidc.UserInfo
		wantStatus  int
	}{
		{
			name:       "unset allows any user",
			idClaims:   map[string]any{"sub": "someone"},
			wantStatus: http.StatusFound,
		},
		{
			name:        "subject match",
			allowedUser: "abc-123",
			idClaims:    map[string]any{"sub": "abc-123"},
			wantStatus:  http.StatusFound,
		},
		{
			name:        "subject is case-sensitive",
			allowedUser: "ABC-123",
			idClaims:    map[string]any{"sub": "abc-123"},
			wantStatus:  http.StatusForbidden,
		},
		{
			name:        "other subject rejected",
			allowedUser: "abc-123",
			idClaims:    map[string]any{"sub": "intruder"},
			wantStatus:  http.StatusForbidden,
		},
		{
			name:        "verified email match ignores case",
			allowedUser: "Me@Example.com",
			idClaims:    map[string]any{"sub": "abc-123", "email": "me@example.com", "email_verified": true},
			wantStatus:  http.StatusFound,
		},
		{
			name:        "email_verified sent as string",
			allowedUser: "me@example.com",
			idClaims:    map[string]any{"sub": "abc-123", "email": "me@example.com", "email_verified": "true"},
			wantStatus:  http.StatusFound,
		},
		{
			name:        "unverified email rejected",
			allowedUser: "me@example.com",
			idClaims:    map[string]any{"sub": "abc-123", "email": "me@example.com", "email_verified": false},
			wantStatus:  http.StatusForbidden,
		},
		{
			name:        "missing email_verified rejected",
			allowedUser: "me@example.com",
			idClaims:    map[string]any{"sub": "abc-123", "email": "me@example.com"},
			wantStatus:  http.StatusForbidden,
		},
		{
			name:        "verified email from userinfo",
			allowedUser: "me@example.com",
			idClaims:    map[string]any{"sub": "abc-123"},
			userInfo:    &oidc.UserInfo{Subject: "abc-123", Email: "me@example.com", EmailVerified: true},
			wantStatus:  http.StatusFound,
		},
		{
			name:        "unverified userinfo email overrides verified ID token email",
			allowedUser: "me@example.com",
			idClaims:    map[string]any{"sub": "abc-123", "email": "old@example.com", "email_verified": true},
			userInfo:    &oidc.UserInfo{Subject: "abc-123", Email: "me@example.com", EmailVerified: false},
			wantStatus:  http.StatusForbidden,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &domain.Config{BaseURL: "/", OIDCEnabled: true, OIDCAllowedUser: tt.allowedUser}

			rec, sm, token := runOIDCCallback(t, cfg, tt.idClaims, tt.userInfo)
			assert.Equal(t, tt.wantStatus, rec.Code)

			ctx, err := sm.Load(t.Context(), token)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus == http.StatusFound, sm.GetBool(ctx, "authenticated"))
		})
	}
}

func TestIsAuthenticatedRechecksOIDCAllowedUser(t *testing.T) {
	tests := []struct {
		name        string
		authMethod  string
		subject     string
		email       string
		allowedUser string
		wantStatus  int
	}{
		{name: "unset allows existing session", authMethod: "oidc", subject: "someone", wantStatus: http.StatusOK},
		{name: "subject still allowed", authMethod: "oidc", subject: "abc-123", allowedUser: "abc-123", wantStatus: http.StatusOK},
		{name: "verified email still allowed", authMethod: "oidc", subject: "abc-123", email: "me@example.com", allowedUser: "ME@example.com", wantStatus: http.StatusOK},
		{name: "allowed user changed", authMethod: "oidc", subject: "abc-123", allowedUser: "other", wantStatus: http.StatusForbidden},
		{name: "session from before the setting existed", authMethod: "oidc", allowedUser: "abc-123", wantStatus: http.StatusForbidden},
		{name: "password session unaffected", authMethod: "password", allowedUser: "abc-123", wantStatus: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := scs.New()
			s := &Server{
				log:            zerolog.Nop(),
				config:         &config.AppConfig{Config: &domain.Config{OIDCAllowedUser: tt.allowedUser}},
				sessionManager: sm,
			}

			ctx, err := sm.Load(t.Context(), "")
			require.NoError(t, err)
			sm.Put(ctx, "authenticated", true)
			sm.Put(ctx, "auth_method", tt.authMethod)
			if tt.subject != "" {
				sm.Put(ctx, "oidc_subject", tt.subject)
			}
			if tt.email != "" {
				sm.Put(ctx, "oidc_verified_email", tt.email)
			}
			token, _, err := sm.Commit(ctx)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
			req.AddCookie(&http.Cookie{Name: sm.Cookie.Name, Value: token})
			rec := httptest.NewRecorder()
			sm.LoadAndSave(s.IsAuthenticated(okHandler())).ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}
