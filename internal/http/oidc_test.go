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
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/alexedwards/scs/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
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

			next := sessions.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sessions.Put(r.Context(), "authenticated", tt.authenticated)
				handler.getConfig(w, r)
			}))
			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/oidc/config", nil))
			require.Equal(t, http.StatusOK, rec.Code)
			assert.NotContains(t, rec.Body.String(), "pkceVerifier")

			var response GetConfigResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, tt.wantRedirect, response.AutoRedirect)
			authURL, err := url.Parse(response.AuthorizationURL)
			require.NoError(t, err)
			require.NotEmpty(t, response.State)
			assert.Equal(t, response.State, authURL.Query().Get("state"))

			login, pending := handler.pendingLogins.Get(response.State)
			if tt.authenticated {
				assert.False(t, pending)
				return
			}

			require.True(t, pending)
			if tt.pkce {
				require.NotEmpty(t, login.verifier)
				challenge := sha256.Sum256([]byte(login.verifier))
				assert.Equal(t, base64.RawURLEncoding.EncodeToString(challenge[:]), authURL.Query().Get("code_challenge"))
				assert.Equal(t, "S256", authURL.Query().Get("code_challenge_method"))
			} else {
				assert.Empty(t, login.verifier)
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

			target := "/api/auth/oidc/config"
			if tt.redirect != "" {
				target += "?redirect=" + url.QueryEscape(tt.redirect)
			}
			rec := httptest.NewRecorder()
			sessions.LoadAndSave(http.HandlerFunc(handler.getConfig)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
			require.Equal(t, http.StatusOK, rec.Code)

			var response GetConfigResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			login, pending := handler.pendingLogins.Get(response.State)
			require.True(t, pending)
			assert.Equal(t, tt.want, login.target)
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

// oidcProviderMock signs real ID tokens so the callback runs the go-oidc verifier
// and claims decoding instead of a stubbed token.
type oidcProviderMock struct {
	oidcConfigServiceMock
	key      *rsa.PrivateKey
	verifier *oidc.IDTokenVerifier
}

const oidcTestIssuer = "https://auth.example.com"

func newOIDCProviderMock(t *testing.T) *oidcProviderMock {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keySet := &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&key.PublicKey}}

	return &oidcProviderMock{
		key:      key,
		verifier: oidc.NewVerifier(oidcTestIssuer, keySet, &oidc.Config{ClientID: "autobrr"}),
	}
}

func (p *oidcProviderMock) OAuthExchange(_ context.Context, _ string, _ ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(map[string]any{
		"iss":                oidcTestIssuer,
		"aud":                "autobrr",
		"sub":                "user-1",
		"preferred_username": "alice",
		"iat":                time.Now().Unix(),
		"exp":                time.Now().Add(time.Minute).Unix(),
	})
	if err != nil {
		return nil, err
	}

	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, p.key, crypto.SHA256, digest[:])
	if err != nil {
		return nil, err
	}

	token := &oauth2.Token{AccessToken: "access-token", TokenType: "Bearer"}

	return token.WithExtra(map[string]any{"id_token": signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)}), nil
}

func (p *oidcProviderMock) VerifyIDToken(ctx context.Context, idToken string) (*oidc.IDToken, error) {
	return p.verifier.Verify(ctx, idToken)
}

func (p *oidcProviderMock) UserInfo(_ context.Context, _ oauth2.TokenSource) (*oidc.UserInfo, error) {
	return nil, errors.New("userinfo not offered")
}

// oidcLoginFlow drives the config and callback routes the way a browser does,
// keeping cookies in a jar shared by every login it starts.
type oidcLoginFlow struct {
	app     http.Handler
	cookies map[string]*http.Cookie
}

func newOIDCLoginFlow(t *testing.T, baseURL string) *oidcLoginFlow {
	sessions := scs.New()
	sessions.Cookie.Path = baseURL
	handler := NewOIDCHandler(encoder{}, zerolog.Nop(), &domain.Config{OIDCEnabled: true, BaseURL: baseURL}, sessions, newOIDCProviderMock(t), func(*http.Request) string { return "" })
	router := chi.NewRouter()
	router.Route("/oidc", handler.Routes)

	return &oidcLoginFlow{app: sessions.LoadAndSave(router), cookies: map[string]*http.Cookie{}}
}

func (f *oidcLoginFlow) do(t *testing.T, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for _, c := range f.cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	f.app.ServeHTTP(rec, req)

	for _, c := range rec.Result().Cookies() {
		if c.MaxAge < 0 {
			delete(f.cookies, c.Name)
			continue
		}
		f.cookies[c.Name] = c
	}

	return rec
}

// start begins a login without the browser's cookies, like a tab whose session expired.
func (f *oidcLoginFlow) start(t *testing.T, redirect string) string {
	target := "/oidc/config"
	if redirect != "" {
		target += "?redirect=" + url.QueryEscape(redirect)
	}

	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	f.app.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	for _, c := range rec.Result().Cookies() {
		f.cookies[c.Name] = c
	}

	var config GetConfigResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &config))

	return config.State
}

func (f *oidcLoginFlow) callback(t *testing.T, state string) *httptest.ResponseRecorder {
	return f.do(t, "/oidc/callback?code=code&state="+url.QueryEscape(state))
}

func TestOIDCCallbackReturnsToRequestedPage(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name     string
		redirect string
		want     string
	}{
		{name: "requested page", redirect: "/autobrr/filters/12?tab=actions", want: "/autobrr/filters/12?tab=actions"},
		{name: "no requested page", want: "/autobrr/"},
		{name: "page outside base URL", redirect: "/elsewhere", want: "/autobrr/"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			flow := newOIDCLoginFlow(t, "/autobrr/")
			state := flow.start(t, tt.redirect)

			rec := flow.callback(t, state)
			require.Equal(t, http.StatusFound, rec.Code, rec.Body.String())
			assert.Equal(t, tt.want, rec.Header().Get("Location"))
		})
	}
}

func TestOIDCCallbackConcurrentLogins(t *testing.T) {
	t.Parallel()
	flow := newOIDCLoginFlow(t, "/autobrr/")
	filters := flow.start(t, "/autobrr/filters")
	releases := flow.start(t, "/autobrr/releases")
	settings := flow.start(t, "/autobrr/settings")

	for state, want := range map[string]string{releases: "/autobrr/releases", settings: "/autobrr/settings", filters: "/autobrr/filters"} {
		rec := flow.callback(t, state)
		require.Equal(t, http.StatusFound, rec.Code, rec.Body.String())
		assert.Equal(t, want, rec.Header().Get("Location"))
	}
}

func TestOIDCCallbackRejectsForeignAndReusedState(t *testing.T) {
	t.Parallel()

	t.Run("state started in another browser", func(t *testing.T) {
		t.Parallel()
		attacker := newOIDCLoginFlow(t, "/")
		state := attacker.start(t, "")
		attacker.cookies = map[string]*http.Cookie{}

		assert.Equal(t, http.StatusBadRequest, attacker.callback(t, state).Code)
	})

	t.Run("state used twice", func(t *testing.T) {
		t.Parallel()
		flow := newOIDCLoginFlow(t, "/")
		state := flow.start(t, "")
		stateCookie := flow.cookies[stateCookieName(state)]

		require.Equal(t, http.StatusFound, flow.callback(t, state).Code)

		flow.cookies[stateCookie.Name] = stateCookie
		assert.Equal(t, http.StatusBadRequest, flow.callback(t, state).Code)
	})

	t.Run("unknown state", func(t *testing.T) {
		t.Parallel()
		flow := newOIDCLoginFlow(t, "/")

		assert.Equal(t, http.StatusBadRequest, flow.callback(t, "unknown").Code)
	})
}
