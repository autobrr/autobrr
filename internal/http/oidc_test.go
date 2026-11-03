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
	"fmt"
	"maps"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/config"
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
		{name: "redirect at the limit", baseURL: "/", redirect: "/filters?q=" + strings.Repeat("a", oidcMaxRedirectLength-len("/filters?q=")), want: "/filters?q=" + strings.Repeat("a", oidcMaxRedirectLength-len("/filters?q="))},
		{name: "redirect longer than the limit", baseURL: "/", redirect: "/filters?q=" + strings.Repeat("a", oidcMaxRedirectLength)},
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

// oidcProviderMock runs the token exchange against a local token endpoint and
// signs real ID tokens, so the callback goes through oauth2, the go-oidc verifier
// and claims decoding instead of stubs.
type oidcProviderMock struct {
	oidcConfigServiceMock
	t        *testing.T
	key      *rsa.PrivateKey
	verifier *oidc.IDTokenVerifier
	tokenURL string

	// claims are added to every ID token, overriding the defaults
	claims      map[string]any
	userInfo    *oidc.UserInfo
	userInfoErr error

	m             sync.Mutex
	codeVerifiers map[string]string
}

const oidcTestIssuer = "https://auth.example.com"

func newOIDCProviderMock(t *testing.T) *oidcProviderMock {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	p := &oidcProviderMock{
		oidcConfigServiceMock: oidcConfigServiceMock{pkce: true},
		t:                     t,
		key:                   key,
		verifier:              oidc.NewVerifier(oidcTestIssuer, &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&key.PublicKey}}, &oidc.Config{ClientID: "autobrr"}),
		userInfoErr:           errors.New("userinfo not offered"),
		codeVerifiers:         map[string]string{},
	}

	tokenServer := httptest.NewServer(http.HandlerFunc(p.token))
	t.Cleanup(tokenServer.Close)
	p.tokenURL = tokenServer.URL

	return p
}

func (p *oidcProviderMock) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p.m.Lock()
	p.codeVerifiers[r.PostForm.Get("code")] = r.PostForm.Get("code_verifier")
	p.m.Unlock()

	idToken, err := p.signIDToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"access_token": "access-token", "token_type": "Bearer", "id_token": idToken}); err != nil {
		p.t.Errorf("could not write token response: %v", err)
	}
}

func (p *oidcProviderMock) codeVerifier(code string) string {
	p.m.Lock()
	defer p.m.Unlock()

	return p.codeVerifiers[code]
}

func (p *oidcProviderMock) signIDToken() (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}

	claims := map[string]any{
		"iss":                oidcTestIssuer,
		"aud":                "autobrr",
		"sub":                "user-1",
		"preferred_username": "alice",
		"iat":                time.Now().Unix(),
		"exp":                time.Now().Add(time.Minute).Unix(),
	}
	maps.Copy(claims, p.claims)

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, p.key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (p *oidcProviderMock) OAuthExchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	cfg := oauth2.Config{ClientID: "autobrr", Endpoint: oauth2.Endpoint{TokenURL: p.tokenURL, AuthStyle: oauth2.AuthStyleInParams}}

	return cfg.Exchange(ctx, code, opts...)
}

func (p *oidcProviderMock) VerifyIDToken(ctx context.Context, idToken string) (*oidc.IDToken, error) {
	return p.verifier.Verify(ctx, idToken)
}

func (p *oidcProviderMock) UserInfo(_ context.Context, _ oauth2.TokenSource) (*oidc.UserInfo, error) {
	return p.userInfo, p.userInfoErr
}

const oidcTestOrigin = "https://autobrr.example.com"

// oidcLoginFlow drives the config and callback routes the way a browser behind
// a TLS-terminating proxy does: a cookie jar applies each cookie's Path and
// Secure attributes, and requests carry X-Forwarded-Proto: https.
type oidcLoginFlow struct {
	app      http.Handler
	handler  *OIDCHandler
	provider *oidcProviderMock
	sessions *scs.SessionManager
	jar      *cookiejar.Jar
	routes   string
}

func newOIDCLoginFlow(t *testing.T, baseURL string) *oidcLoginFlow {
	routes := strings.TrimSuffix(baseURL, "/") + "/api/auth/oidc"
	sessions := scs.New()
	sessions.Cookie.Path = baseURL
	provider := newOIDCProviderMock(t)
	handler := NewOIDCHandler(encoder{}, zerolog.Nop(), &domain.Config{
		OIDCEnabled:     true,
		BaseURL:         baseURL,
		OIDCRedirectURL: oidcTestOrigin + routes + "/callback",
	}, sessions, provider, func(*http.Request) string { return "" })
	router := chi.NewRouter()
	router.Route(routes, handler.Routes)
	router.Get(routes+"/session", func(w http.ResponseWriter, r *http.Request) {
		if !sessions.GetBool(r.Context(), "authenticated") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	return &oidcLoginFlow{app: sessions.LoadAndSave(router), handler: handler, provider: provider, sessions: sessions, jar: jar, routes: routes}
}

func (f *oidcLoginFlow) do(t *testing.T, target string, withCookies bool) *httptest.ResponseRecorder {
	u, err := url.Parse(oidcTestOrigin + target)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, u.String(), nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	if withCookies {
		for _, c := range f.jar.Cookies(u) {
			req.AddCookie(c)
		}
	}
	rec := httptest.NewRecorder()
	f.app.ServeHTTP(rec, req)
	f.jar.SetCookies(u, rec.Result().Cookies())

	return rec
}

// start begins a login without sending cookies, like a tab whose session expired,
// and returns its state and authorization URL.
func (f *oidcLoginFlow) start(t *testing.T, redirect string) (string, *url.URL) {
	target := f.routes + "/config"
	if redirect != "" {
		target += "?redirect=" + url.QueryEscape(redirect)
	}

	rec := f.do(t, target, false)
	require.Equal(t, http.StatusOK, rec.Code)

	var config GetConfigResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &config))
	authURL, err := url.Parse(config.AuthorizationURL)
	require.NoError(t, err)

	return config.State, authURL
}

func (f *oidcLoginFlow) callbackURL(t *testing.T) *url.URL {
	u, err := url.Parse(oidcTestOrigin + f.routes + "/callback")
	require.NoError(t, err)

	return u
}

func (f *oidcLoginFlow) callback(t *testing.T, state string) *httptest.ResponseRecorder {
	return f.do(t, f.routes+"/callback?code="+url.QueryEscape("code-"+state)+"&state="+url.QueryEscape(state), true)
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
			state, _ := flow.start(t, tt.redirect)

			rec := flow.callback(t, state)
			require.Equal(t, http.StatusFound, rec.Code, rec.Body.String())
			assert.Equal(t, tt.want, rec.Header().Get("Location"))
		})
	}
}

func TestOIDCCallbackSendsPKCEVerifier(t *testing.T) {
	t.Parallel()
	flow := newOIDCLoginFlow(t, "/")
	state, authURL := flow.start(t, "")
	require.Equal(t, "S256", authURL.Query().Get("code_challenge_method"))

	require.Equal(t, http.StatusFound, flow.callback(t, state).Code)

	verifier := flow.provider.codeVerifier("code-" + state)
	require.NotEmpty(t, verifier)
	challenge := sha256.Sum256([]byte(verifier))
	assert.Equal(t, authURL.Query().Get("code_challenge"), base64.RawURLEncoding.EncodeToString(challenge[:]))
}

func TestOIDCCallbackInterleavedLogins(t *testing.T) {
	t.Parallel()
	flow := newOIDCLoginFlow(t, "/autobrr/")
	filters, filtersURL := flow.start(t, "/autobrr/filters")
	releases, releasesURL := flow.start(t, "/autobrr/releases")
	settings, settingsURL := flow.start(t, "/autobrr/settings")
	challenges := map[string]string{
		filters:  filtersURL.Query().Get("code_challenge"),
		releases: releasesURL.Query().Get("code_challenge"),
		settings: settingsURL.Query().Get("code_challenge"),
	}
	require.Len(t, map[string]bool{challenges[filters]: true, challenges[releases]: true, challenges[settings]: true}, 3, "every login gets its own PKCE verifier")

	for state, want := range map[string]string{releases: "/autobrr/releases", settings: "/autobrr/settings", filters: "/autobrr/filters"} {
		rec := flow.callback(t, state)
		require.Equal(t, http.StatusFound, rec.Code, rec.Body.String())
		assert.Equal(t, want, rec.Header().Get("Location"))

		sum := sha256.Sum256([]byte(flow.provider.codeVerifier("code-" + state)))
		assert.Equal(t, challenges[state], base64.RawURLEncoding.EncodeToString(sum[:]))
	}
}

// TestOIDCCallbackSimultaneousLogins sends every login's requests at the same time,
// so the race detector sees the pending-login store, cookies and sessions under
// overlapping requests. The goroutines only record results: require must run on
// the test goroutine.
func TestOIDCCallbackSimultaneousLogins(t *testing.T) {
	t.Parallel()
	flow := newOIDCLoginFlow(t, "/autobrr/")
	targets := []string{"/autobrr/filters", "/autobrr/releases", "/autobrr/settings", "/autobrr/logs", "/autobrr/"}

	type result struct {
		configCode   int
		callbackCode int
		state        string
		location     string
		challenge    string
	}
	results := make([]result, len(targets))

	var wg sync.WaitGroup
	for i, target := range targets {
		wg.Go(func() {
			u, _ := url.Parse(oidcTestOrigin + flow.routes + "/config?redirect=" + url.QueryEscape(target))
			req := httptest.NewRequest(http.MethodGet, u.String(), nil)
			req.Header.Set("X-Forwarded-Proto", "https")
			rec := httptest.NewRecorder()
			flow.app.ServeHTTP(rec, req)
			flow.jar.SetCookies(u, rec.Result().Cookies())

			var config GetConfigResponse
			_ = json.Unmarshal(rec.Body.Bytes(), &config)
			authURL, _ := url.Parse(config.AuthorizationURL)
			results[i] = result{configCode: rec.Code, state: config.State, challenge: authURL.Query().Get("code_challenge")}

			cu, _ := url.Parse(oidcTestOrigin + flow.routes + "/callback?code=" + url.QueryEscape("code-"+config.State) + "&state=" + url.QueryEscape(config.State))
			req = httptest.NewRequest(http.MethodGet, cu.String(), nil)
			req.Header.Set("X-Forwarded-Proto", "https")
			for _, c := range flow.jar.Cookies(cu) {
				req.AddCookie(c)
			}
			callback := httptest.NewRecorder()
			flow.app.ServeHTTP(callback, req)
			flow.jar.SetCookies(cu, callback.Result().Cookies())
			results[i].callbackCode = callback.Code
			results[i].location = callback.Header().Get("Location")
		})
	}
	wg.Wait()

	challenges := map[string]bool{}
	for i, res := range results {
		require.Equal(t, http.StatusOK, res.configCode)
		assert.Equal(t, http.StatusFound, res.callbackCode, targets[i])
		assert.Equal(t, targets[i], res.location)

		sum := sha256.Sum256([]byte(flow.provider.codeVerifier("code-" + res.state)))
		assert.Equal(t, res.challenge, base64.RawURLEncoding.EncodeToString(sum[:]))
		challenges[res.challenge] = true
	}
	assert.Len(t, challenges, len(targets))

	assert.Equal(t, http.StatusNoContent, flow.do(t, flow.routes+"/session", true).Code, "the browser ends up with an authenticated session")
	assert.Empty(t, flow.handler.pendingLogins.GetKeys())
}

func TestOIDCStateCookie(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name       string
		https      bool
		wantSecure bool
	}{
		{name: "behind TLS proxy", https: true, wantSecure: true},
		{name: "plain HTTP"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			flow := newOIDCLoginFlow(t, "/autobrr/")
			req := httptest.NewRequest(http.MethodGet, oidcTestOrigin+flow.routes+"/config", nil)
			if tt.https {
				req.Header.Set("X-Forwarded-Proto", "https")
			}
			rec := httptest.NewRecorder()
			flow.app.ServeHTTP(rec, req)

			var config GetConfigResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &config))

			var stateCookie *http.Cookie
			for _, c := range rec.Result().Cookies() {
				if c.Name == stateCookieName(config.State) {
					stateCookie = c
				}
			}
			require.NotNil(t, stateCookie)
			assert.Equal(t, "/autobrr/api/auth/oidc/callback", stateCookie.Path)
			assert.Equal(t, tt.wantSecure, stateCookie.Secure)
			assert.Equal(t, int(oidcLoginTimeout.Seconds()), stateCookie.MaxAge)
			assert.True(t, stateCookie.HttpOnly)

			item, ok := flow.handler.pendingLogins.GetItem(config.State)
			require.True(t, ok)
			assert.Equal(t, oidcLoginTimeout, item.GetDuration())
		})
	}
}

func TestOIDCStateCookieOnlyReachesCallback(t *testing.T) {
	t.Parallel()
	flow := newOIDCLoginFlow(t, "/autobrr/")
	state, _ := flow.start(t, "")

	app, err := url.Parse(oidcTestOrigin + "/autobrr/api/filters")
	require.NoError(t, err)
	for _, c := range flow.jar.Cookies(app) {
		assert.NotEqual(t, stateCookieName(state), c.Name)
	}

	names := []string{}
	for _, c := range flow.jar.Cookies(flow.callbackURL(t)) {
		names = append(names, c.Name)
	}
	assert.Contains(t, names, stateCookieName(state))
}

func TestOIDCCallbackRejectsForeignAndReusedState(t *testing.T) {
	t.Parallel()

	t.Run("state started in another browser", func(t *testing.T) {
		t.Parallel()
		attacker := newOIDCLoginFlow(t, "/")
		state, _ := attacker.start(t, "")
		attacker.jar, _ = cookiejar.New(nil)

		assert.Equal(t, http.StatusBadRequest, attacker.callback(t, state).Code)
	})

	t.Run("state cookie with another value", func(t *testing.T) {
		t.Parallel()
		flow := newOIDCLoginFlow(t, "/")
		state, _ := flow.start(t, "")
		flow.jar.SetCookies(flow.callbackURL(t), []*http.Cookie{{Name: stateCookieName(state), Value: "other", Path: flow.callbackURL(t).Path}})

		assert.Equal(t, http.StatusBadRequest, flow.callback(t, state).Code)
	})

	t.Run("state used twice", func(t *testing.T) {
		t.Parallel()
		flow := newOIDCLoginFlow(t, "/")
		state, _ := flow.start(t, "")
		stateCookie := &http.Cookie{Name: stateCookieName(state), Value: state, Path: flow.callbackURL(t).Path}

		require.Equal(t, http.StatusFound, flow.callback(t, state).Code)

		flow.jar.SetCookies(flow.callbackURL(t), []*http.Cookie{stateCookie})
		assert.Equal(t, http.StatusBadRequest, flow.callback(t, state).Code)
	})

	t.Run("expired login", func(t *testing.T) {
		t.Parallel()
		flow := newOIDCLoginFlow(t, "/")
		state, _ := flow.start(t, "")
		flow.handler.pendingLogins.Set(state, pendingOIDCLogin{}, time.Millisecond)
		time.Sleep(20 * time.Millisecond)

		assert.Equal(t, http.StatusBadRequest, flow.callback(t, state).Code)
	})

	t.Run("unknown state", func(t *testing.T) {
		t.Parallel()
		flow := newOIDCLoginFlow(t, "/")

		assert.Equal(t, http.StatusBadRequest, flow.callback(t, "unknown").Code)
	})
}

func TestOIDCFinishLoginTakesStateOnce(t *testing.T) {
	t.Parallel()
	flow := newOIDCLoginFlow(t, "/")
	state, _ := flow.start(t, "")

	var wg sync.WaitGroup
	var taken atomic.Int32
	for range 50 {
		wg.Go(func() {
			req := httptest.NewRequest(http.MethodGet, "/callback", nil)
			req.AddCookie(&http.Cookie{Name: stateCookieName(state), Value: state})
			if _, ok := flow.handler.finishLogin(httptest.NewRecorder(), req, state); ok {
				taken.Add(1)
			}
		})
	}
	wg.Wait()

	assert.Equal(t, int32(1), taken.Load())
}

func TestOIDCPendingLoginsAreCapped(t *testing.T) {
	t.Parallel()
	flow := newOIDCLoginFlow(t, "/")
	first, _ := flow.start(t, "")

	for i := range oidcMaxPendingLogins {
		req := httptest.NewRequest(http.MethodGet, "/config", nil)
		flow.handler.startLogin(httptest.NewRecorder(), req, fmt.Sprintf("state-%d", i), pendingOIDCLogin{})
	}

	assert.Len(t, flow.handler.pendingLogins.GetKeys(), oidcMaxPendingLogins)
	_, kept := flow.handler.pendingLogins.Get(first)
	assert.False(t, kept, "the oldest login is evicted first")
	_, kept = flow.handler.pendingLogins.Get(fmt.Sprintf("state-%d", oidcMaxPendingLogins-1))
	assert.True(t, kept)
}

func TestOIDCStateCookiePath(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name        string
		baseURL     string
		redirectURL string
		want        string
	}{
		{name: "callback under base URL", baseURL: "/autobrr/", redirectURL: "https://autobrr.example.com/autobrr/api/auth/oidc/callback", want: "/autobrr/api/auth/oidc/callback"},
		{name: "callback at root", baseURL: "/", redirectURL: "https://autobrr.example.com/api/auth/oidc/callback", want: "/api/auth/oidc/callback"},
		{name: "callback outside base URL", baseURL: "/autobrr/", redirectURL: "https://autobrr.example.com/oidc/callback", want: "/autobrr/"},
		{name: "base URL without trailing slash", baseURL: "/autobrr", redirectURL: "https://autobrr.example.com/autobrrx/callback", want: "/autobrr"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			sessions := scs.New()
			sessions.Cookie.Path = tt.baseURL
			handler := NewOIDCHandler(encoder{}, zerolog.Nop(), &domain.Config{OIDCEnabled: true, BaseURL: tt.baseURL, OIDCRedirectURL: tt.redirectURL}, sessions, &oidcConfigServiceMock{}, func(*http.Request) string { return "" })

			assert.Equal(t, tt.want, handler.stateCookiePath())
		})
	}
}

// session loads the session this browser holds after the login flow.
func (f *oidcLoginFlow) session(t *testing.T) context.Context {
	var token string
	for _, c := range f.jar.Cookies(f.callbackURL(t)) {
		if c.Name == f.sessions.Cookie.Name {
			token = c.Value
		}
	}

	ctx, err := f.sessions.Load(t.Context(), token)
	require.NoError(t, err)

	return ctx
}

func TestOIDCCallbackAllowedUser(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name           string
		allowedSubject string
		allowedEmail   string
		claims         map[string]any
		userInfo       *oidc.UserInfo
		userInfoErr    error
		wantStatus     int
		wantEmail      string
	}{
		{
			name:       "unset allows any user",
			claims:     map[string]any{"sub": "someone"},
			wantStatus: http.StatusFound,
		},
		{
			name:           "subject match",
			allowedSubject: "abc-123",
			claims:         map[string]any{"sub": "abc-123"},
			wantStatus:     http.StatusFound,
		},
		{
			name:           "subject is case-sensitive",
			allowedSubject: "ABC-123",
			claims:         map[string]any{"sub": "abc-123"},
			wantStatus:     http.StatusForbidden,
		},
		{
			name:           "other subject rejected",
			allowedSubject: "abc-123",
			claims:         map[string]any{"sub": "intruder"},
			wantStatus:     http.StatusForbidden,
		},
		{
			name:           "subject setting ignores a matching verified email",
			allowedSubject: "me@example.com",
			claims:         map[string]any{"sub": "intruder", "email": "me@example.com", "email_verified": true},
			wantStatus:     http.StatusForbidden,
		},
		{
			name:         "email setting ignores a subject that looks like the email",
			allowedEmail: "me@example.com",
			claims:       map[string]any{"sub": "me@example.com"},
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "verified email match ignores case",
			allowedEmail: "Me@Example.com",
			claims:       map[string]any{"sub": "abc-123", "email": "me@example.com", "email_verified": true},
			wantStatus:   http.StatusFound,
			wantEmail:    "me@example.com",
		},
		{
			name:         "email_verified sent as string",
			allowedEmail: "me@example.com",
			claims:       map[string]any{"sub": "abc-123", "email": "me@example.com", "email_verified": "true"},
			wantStatus:   http.StatusFound,
			wantEmail:    "me@example.com",
		},
		{
			name:         "unverified email rejected",
			allowedEmail: "me@example.com",
			claims:       map[string]any{"sub": "abc-123", "email": "me@example.com", "email_verified": false},
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "missing email_verified rejected",
			allowedEmail: "me@example.com",
			claims:       map[string]any{"sub": "abc-123", "email": "me@example.com"},
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "verified email from userinfo",
			allowedEmail: "me@example.com",
			claims:       map[string]any{"sub": "abc-123"},
			userInfo:     &oidc.UserInfo{Subject: "abc-123", Email: "me@example.com", EmailVerified: true},
			wantStatus:   http.StatusFound,
			wantEmail:    "me@example.com",
		},
		{
			name:         "unverified userinfo email overrides verified ID token email",
			allowedEmail: "me@example.com",
			claims:       map[string]any{"sub": "abc-123", "email": "old@example.com", "email_verified": true},
			userInfo:     &oidc.UserInfo{Subject: "abc-123", Email: "me@example.com", EmailVerified: false},
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "userinfo for another subject cannot supply the email",
			allowedEmail: "me@example.com",
			claims:       map[string]any{"sub": "intruder"},
			userInfo:     &oidc.UserInfo{Subject: "abc-123", Email: "me@example.com", EmailVerified: true},
			wantStatus:   http.StatusForbidden,
		},
		{
			name:         "userinfo for another subject ignored, ID token email still used",
			allowedEmail: "me@example.com",
			claims:       map[string]any{"sub": "abc-123", "email": "me@example.com", "email_verified": true},
			userInfo:     &oidc.UserInfo{Subject: "intruder", Email: "other@example.com", EmailVerified: false},
			wantStatus:   http.StatusFound,
			wantEmail:    "me@example.com",
		},
		{
			name:         "userinfo failure reported when email could not be verified",
			allowedEmail: "me@example.com",
			claims:       map[string]any{"sub": "abc-123"},
			userInfoErr:  errors.New("provider unavailable: internal.example.lan"),
			wantStatus:   http.StatusBadGateway,
		},
		{
			name:         "userinfo failure with a verified non-matching ID token email",
			allowedEmail: "me@example.com",
			claims:       map[string]any{"sub": "abc-123", "email": "other@example.com", "email_verified": true},
			userInfoErr:  errors.New("provider unavailable: internal.example.lan"),
			wantStatus:   http.StatusBadGateway,
		},
		{
			name:           "userinfo failure does not affect subject setting",
			allowedSubject: "abc-123",
			claims:         map[string]any{"sub": "intruder"},
			userInfoErr:    errors.New("provider unavailable"),
			wantStatus:     http.StatusForbidden,
		},
		{
			name:           "userinfo failure does not block subject match",
			allowedSubject: "abc-123",
			claims:         map[string]any{"sub": "abc-123"},
			userInfoErr:    errors.New("provider unavailable"),
			wantStatus:     http.StatusFound,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			flow := newOIDCLoginFlow(t, "/")
			flow.handler.config.OIDCAllowedSubject = tt.allowedSubject
			flow.handler.config.OIDCAllowedEmail = tt.allowedEmail
			flow.provider.claims = tt.claims
			flow.provider.userInfo = tt.userInfo
			flow.provider.userInfoErr = tt.userInfoErr
			state, _ := flow.start(t, "")

			rec := flow.callback(t, state)
			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			if tt.userInfoErr != nil {
				assert.NotContains(t, rec.Body.String(), tt.userInfoErr.Error(), "provider errors stay in the log")
				assert.NotContains(t, rec.Body.String(), "internal.example.lan", "provider host stays in the log")
			}

			ctx := flow.session(t)
			assert.Equal(t, tt.wantStatus == http.StatusFound, flow.sessions.GetBool(ctx, "authenticated"))
			if tt.wantStatus == http.StatusFound {
				assert.Equal(t, "oidc", flow.sessions.GetString(ctx, "auth_method"))
				assert.Equal(t, tt.claims["sub"], flow.sessions.GetString(ctx, "oidc_subject"))
				assert.Equal(t, tt.wantEmail, flow.sessions.GetString(ctx, "oidc_verified_email"))
			}
		})
	}
}

func TestIsAuthenticatedRechecksOIDCAllowedUser(t *testing.T) {
	tests := []struct {
		name           string
		authMethod     string
		subject        string
		email          string
		allowedSubject string
		allowedEmail   string
		wantStatus     int
	}{
		{name: "unset allows existing session", authMethod: "oidc", subject: "someone", wantStatus: http.StatusOK},
		{name: "subject still allowed", authMethod: "oidc", subject: "abc-123", allowedSubject: "abc-123", wantStatus: http.StatusOK},
		{name: "verified email still allowed", authMethod: "oidc", subject: "abc-123", email: "me@example.com", allowedEmail: "ME@example.com", wantStatus: http.StatusOK},
		{name: "allowed subject changed", authMethod: "oidc", subject: "abc-123", allowedSubject: "other", wantStatus: http.StatusForbidden},
		{name: "switched from subject to email", authMethod: "oidc", subject: "abc-123", allowedEmail: "me@example.com", wantStatus: http.StatusForbidden},
		{name: "session from before the setting existed", authMethod: "oidc", allowedSubject: "abc-123", wantStatus: http.StatusForbidden},
		{name: "password session unaffected", authMethod: "password", allowedSubject: "abc-123", wantStatus: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := scs.New()
			s := &Server{
				log:            zerolog.Nop(),
				config:         &config.AppConfig{Config: &domain.Config{OIDCAllowedSubject: tt.allowedSubject, OIDCAllowedEmail: tt.allowedEmail}},
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

			// allowed requests may renew the token, so only the destroyed
			// session is still reachable under the original one
			if tt.wantStatus == http.StatusForbidden {
				ctx, err := sm.Load(t.Context(), token)
				require.NoError(t, err)
				assert.False(t, sm.GetBool(ctx, "authenticated"))
			}
		})
	}
}
