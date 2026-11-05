// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/v2/memstore"
	"github.com/autobrr/go-cache/ttlcache"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type authServiceMock struct {
	users    map[string]*domain.User
	loginErr error
}

func (a authServiceMock) GetUserCount(ctx context.Context) (int, error) {
	return len(a.users), nil
}

func (a authServiceMock) Login(ctx context.Context, username, password string) (*domain.User, error) {
	if a.loginErr != nil {
		return nil, a.loginErr
	}

	u, ok := a.users[username]
	if !ok {
		return nil, domain.ErrInvalidCredentials
	}

	if u.Password != password {
		return nil, domain.ErrInvalidCredentials
	}

	return u, nil
}

func (a authServiceMock) CreateUser(ctx context.Context, req domain.CreateUserRequest) error {
	if req.Username != "" {
		a.users[req.Username] = &domain.User{
			ID:       len(a.users) + 1,
			Username: req.Username,
			Password: req.Password,
		}
	}

	return nil
}

func (a authServiceMock) UpdateUser(ctx context.Context, req domain.UpdateUserRequest) error {
	u, ok := a.users[req.UsernameCurrent]
	if !ok {
		return errors.New("user not found")
	}

	if req.UsernameNew != "" {
		u.Username = req.UsernameNew
	}

	if req.PasswordNew != "" {
		u.Password = req.PasswordNew
	}

	return nil
}

type oidcAuthServiceMock struct {
	unavailable bool
}

func (o *oidcAuthServiceMock) IsEnabled() bool {
	return false
}

func (o *oidcAuthServiceMock) IsReady() bool {
	return !o.unavailable
}

func (o *oidcAuthServiceMock) ValidateConfig() error {
	return nil
}

func (o *oidcAuthServiceMock) UserInfo(ctx context.Context, tokenSource oauth2.TokenSource) (*oidc.UserInfo, error) {
	return nil, nil
}

func (o *oidcAuthServiceMock) VerifyIDToken(ctx context.Context, idToken string) (*oidc.IDToken, error) {
	return nil, nil
}

func (o *oidcAuthServiceMock) GetEndpoint() oauth2.Endpoint {
	return oauth2.Endpoint{}
}

func (o *oidcAuthServiceMock) OAuthExchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return nil, nil
}

func (o *oidcAuthServiceMock) OauthAuthCodeURL(state string, opts ...oauth2.AuthCodeOption) string {
	return ""
}

func (o *oidcAuthServiceMock) SupportsPKCE() bool {
	return false
}

func newHttpTestClient() *http.Client {
	c := &http.Client{}

	jarOptions := &cookiejar.Options{PublicSuffixList: nil}
	jar, err := cookiejar.New(jarOptions)
	if err != nil {
		log.Fatalf("error creating cookiejar: %v", err)
	}

	c.Jar = jar

	return c
}

func setupServer(srv *Server) chi.Router {
	r := chi.NewRouter()
	//r.Use(middleware.Logger)
	r.Use(srv.sessionMiddleware)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	return r
}

func runTestServer(s chi.Router) *httptest.Server {
	return httptest.NewServer(s)
}

func setupAuthHandler() {

}

func TestAuthHandlerLoginMixedSchemes(t *testing.T) {
	srv := NewServer(Deps{
		Log:            zerolog.Nop(),
		Config:         &config.AppConfig{Config: &domain.Config{BaseURL: "/autobrr/"}},
		SessionManager: scs.New(),
	})
	service := authServiceMock{users: map[string]*domain.User{
		"test": {Username: "test", Password: "pass"},
	}}
	handler := newAuthHandler(encoder{}, zerolog.Nop(), srv, srv.config.Config, srv.sessionManager, service, &oidcAuthServiceMock{})
	router := setupServer(srv)
	router.Route("/autobrr/auth", handler.Routes)

	login := func(secure bool, previous *http.Cookie) *http.Cookie {
		t.Helper()

		req := httptest.NewRequest(http.MethodPost, "/autobrr/auth/login", strings.NewReader(`{"username":"test","password":"pass","remember_me":true}`))
		if secure {
			req.Header.Set("X-Forwarded-Proto", "https")
		}
		if previous != nil {
			req.AddCookie(previous)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		cookies := rec.Result().Cookies()
		if !assert.Len(t, cookies, 1) {
			return nil
		}
		cookie := cookies[0]
		assert.Equal(t, secure, cookie.Secure)
		assert.True(t, cookie.HttpOnly)
		assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
		assert.Equal(t, "/autobrr/", cookie.Path)
		return cookie
	}
	validate := func(cookie *http.Cookie, status int) {
		t.Helper()

		req := httptest.NewRequest(http.MethodGet, "/autobrr/auth/validate", nil)
		if cookie != nil && !cookie.Secure {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, status, rec.Code)
	}

	httpCookie := login(false, nil)
	validate(httpCookie, http.StatusOK)
	login(true, nil)
	validate(httpCookie, http.StatusOK)
	newCookie := login(false, httpCookie)
	validate(newCookie, http.StatusOK)
	validate(httpCookie, http.StatusForbidden)

	for _, secure := range []bool{false, true} {
		cookie := login(secure, nil)
		if cookie == nil {
			continue
		}
		req := httptest.NewRequest(http.MethodPost, "/autobrr/auth/logout", nil)
		req.AddCookie(cookie)
		if secure {
			req.Header.Set("X-Forwarded-Proto", "https")
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		cookies := rec.Result().Cookies()
		if assert.Len(t, cookies, 1) {
			assert.Equal(t, secure, cookies[0].Secure)
			assert.Equal(t, -1, cookies[0].MaxAge)
			assert.Equal(t, "/autobrr/", cookies[0].Path)
		}
	}

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() { login(i%2 == 0, nil) })
	}
	wg.Wait()
}

func TestServer_sessionMiddleware(t *testing.T) {
	for _, secure := range []bool{false, true} {
		for _, mode := range []string{"header", "body", "empty", "renew", "destroy", "forbidden"} {
			name := "http_" + mode
			if secure {
				name = "https_" + mode
			}
			t.Run(name, func(t *testing.T) {
				srv := NewServer(Deps{
					Log:            zerolog.Nop(),
					Config:         &config.AppConfig{Config: &domain.Config{BaseURL: "/"}},
					SessionManager: scs.New(),
				})
				handler := srv.sessionMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					http.SetCookie(w, &http.Cookie{Name: "other", Value: "value", Path: "/"})
					srv.sessionManager.Put(r.Context(), "authenticated", true)
					switch mode {
					case "header":
						w.WriteHeader(http.StatusNoContent)
					case "body":
						_, err := w.Write([]byte("OK"))
						assert.NoError(t, err)
					case "renew":
						assert.NoError(t, srv.sessionManager.RenewToken(r.Context()))
					case "destroy":
						assert.NoError(t, srv.sessionManager.Destroy(r.Context()))
					case "forbidden":
						srv.sessionManager.Remove(r.Context(), "authenticated")
						srv.IsAuthenticated(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
							assert.Fail(t, "unauthenticated request reached handler")
						})).ServeHTTP(w, r)
					}
				}))
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				if secure {
					req.Header.Set("X-Forwarded-Proto", "https")
				}
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				cookies := rec.Result().Cookies()
				if !assert.Len(t, cookies, 2) {
					return
				}
				assert.Equal(t, "other", cookies[0].Name)
				assert.False(t, cookies[0].Secure)
				assert.Equal(t, "autobrr_user_session", cookies[1].Name)
				assert.Equal(t, secure, cookies[1].Secure)
				if mode == "destroy" || mode == "forbidden" {
					assert.Equal(t, -1, cookies[1].MaxAge)
					assert.Empty(t, cookies[1].Value)
				}
			})
		}
	}
}

func TestAuthHandlerLogin(t *testing.T) {
	t.Parallel()
	logger := zerolog.Nop()
	encoder := encoder{}
	sessionManager := scs.New()

	service := authServiceMock{
		users: map[string]*domain.User{
			"test": {
				ID:       0,
				Username: "test",
				Password: "pass",
			},
		},
	}

	oidcServiceMock := &oidcAuthServiceMock{}

	server := &Server{
		log:            logger,
		config:         &config.AppConfig{Config: &domain.Config{}},
		sessionManager: sessionManager,
	}

	handler := newAuthHandler(encoder, logger, server, &domain.Config{}, server.sessionManager, service, oidcServiceMock)
	s := setupServer(server)
	s.Route("/auth", handler.Routes)

	testServer := runTestServer(s)
	defer testServer.Close()

	// generate request, here we'll use login as example
	reqBody, err := json.Marshal(map[string]string{
		"username": "test",
		"password": "pass",
	})
	require.NoError(t, err)

	client := newHttpTestClient()

	// make request
	resp, err := client.Post(testServer.URL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	defer resp.Body.Close()

	// check for response, here we'll just check for 204 NoContent
	assert.Equalf(t, http.StatusNoContent, resp.StatusCode, "login handler: unexpected http status")

	assert.NotEmpty(t, resp.Header.Get("Set-Cookie"), "handler returned no cookie")
}

func TestAuthHandlerLoginPerMode(t *testing.T) {
	tests := []struct {
		name   string
		config domain.Config
		want   int
	}{
		{name: "builtin", config: domain.Config{}, want: http.StatusNoContent},
		{name: "oidc with password fallback", config: domain.Config{OIDCEnabled: true}, want: http.StatusNoContent},
		{name: "oidc only", config: domain.Config{OIDCEnabled: true, OIDCDisableBuiltInLogin: true}, want: http.StatusForbidden},
		{name: "disabled", config: domain.Config{AuthDisabled: true, AuthDisabledAcknowledgement: domain.AuthDisabledAcknowledgementValue}, want: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := authServiceMock{users: map[string]*domain.User{"test": {Username: "test", Password: "pass"}}}
			server := &Server{
				log:            zerolog.Nop(),
				config:         &config.AppConfig{Config: &tt.config},
				sessionManager: scs.New(),
			}

			handler := newAuthHandler(encoder{}, zerolog.Nop(), server, &tt.config, server.sessionManager, service, &oidcAuthServiceMock{})
			router := setupServer(server)
			router.Route("/auth", handler.Routes)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"test","password":"pass"}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.want, rec.Code)
			if tt.want == http.StatusForbidden {
				assert.Empty(t, rec.Header().Values("Set-Cookie"))
			} else {
				assert.NotEmpty(t, rec.Header().Values("Set-Cookie"))
			}
		})
	}
}

func TestAuthHandlerValidateOK(t *testing.T) {
	t.Parallel()
	logger := zerolog.Nop()
	encoder := encoder{}
	sessionManager := scs.New()

	service := authServiceMock{
		users: map[string]*domain.User{
			"test": {
				ID:       0,
				Username: "test",
				Password: "pass",
			},
		},
	}

	oidcServiceMock := &oidcAuthServiceMock{}

	server := &Server{
		log:            logger,
		config:         &config.AppConfig{Config: &domain.Config{}},
		sessionManager: sessionManager,
	}

	handler := newAuthHandler(encoder, logger, server, &domain.Config{}, server.sessionManager, service, oidcServiceMock)
	s := setupServer(server)
	s.Route("/auth", handler.Routes)

	testServer := runTestServer(s)
	defer testServer.Close()

	// generate request, here we'll use login as example
	reqBody, err := json.Marshal(map[string]string{
		"username": "test",
		"password": "pass",
	})
	require.NoError(t, err)

	client := newHttpTestClient()

	// make request
	resp, err := client.Post(testServer.URL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	defer resp.Body.Close()

	// check for response, here we'll just check for 204 NoContent
	assert.Equalf(t, http.StatusNoContent, resp.StatusCode, "login handler: bad response")

	assert.NotEmpty(t, resp.Header.Get("Set-Cookie"), "login handler: expected Set-Cookie header")

	// validate token
	resp, err = client.Get(testServer.URL + "/auth/validate")
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equalf(t, http.StatusOK, resp.StatusCode, "validate handler: unexpected http status")

	var caller Caller
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&caller))
	assert.Equal(t, Caller{Method: domain.AuthMethodPassword, Username: "test"}, caller)
}

func TestAuthHandlerValidateBad(t *testing.T) {
	t.Parallel()
	logger := zerolog.Nop()
	encoder := encoder{}
	sessionManager := scs.New()

	service := authServiceMock{
		users: map[string]*domain.User{
			"test": {
				ID:       0,
				Username: "test",
				Password: "pass",
			},
		},
	}

	oidcServiceMock := &oidcAuthServiceMock{}

	server := &Server{
		log:            logger,
		config:         &config.AppConfig{Config: &domain.Config{}},
		sessionManager: sessionManager,
	}

	handler := newAuthHandler(encoder, logger, server, &domain.Config{}, server.sessionManager, service, oidcServiceMock)
	s := setupServer(server)
	s.Route("/auth", handler.Routes)

	testServer := runTestServer(s)
	defer testServer.Close()

	client := newHttpTestClient()

	// validate token
	resp, err := client.Get(testServer.URL + "/auth/validate")
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equalf(t, http.StatusForbidden, resp.StatusCode, "validate handler: unexpected http status")
}

func TestAuthHandlerLoginBad(t *testing.T) {
	t.Parallel()
	logger := zerolog.Nop()
	encoder := encoder{}
	sessionManager := scs.New()

	service := authServiceMock{
		users: map[string]*domain.User{
			"test": {
				ID:       0,
				Username: "test",
				Password: "pass",
			},
		},
	}

	oidcServiceMock := &oidcAuthServiceMock{}

	server := &Server{
		log:            logger,
		config:         &config.AppConfig{Config: &domain.Config{}},
		sessionManager: sessionManager,
	}

	handler := newAuthHandler(encoder, logger, server, &domain.Config{}, server.sessionManager, service, oidcServiceMock)
	s := setupServer(server)
	s.Route("/auth", handler.Routes)

	testServer := runTestServer(s)
	defer testServer.Close()

	// generate request, here we'll use login as example
	reqBody, err := json.Marshal(map[string]string{
		"username": "test",
		"password": "notmypass",
	})
	require.NoError(t, err)

	client := newHttpTestClient()

	// make request
	resp, err := client.Post(testServer.URL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	defer resp.Body.Close()

	// check for response, here we'll just check for 403 Forbidden
	assert.Equalf(t, http.StatusForbidden, resp.StatusCode, "login handler: unexpected http status")
}

func TestAuthHandlerLogout(t *testing.T) {
	logger := zerolog.Nop()
	encoder := encoder{}
	sessionManager := scs.New()

	service := authServiceMock{
		users: map[string]*domain.User{
			"test": {
				ID:       0,
				Username: "test",
				Password: "pass",
			},
		},
	}

	oidcServiceMock := &oidcAuthServiceMock{}

	server := &Server{
		log:            logger,
		config:         &config.AppConfig{Config: &domain.Config{}},
		sessionManager: sessionManager,
	}

	handler := newAuthHandler(encoder, logger, server, &domain.Config{}, server.sessionManager, service, oidcServiceMock)
	s := setupServer(server)
	s.Route("/auth", handler.Routes)

	testServer := runTestServer(s)
	defer testServer.Close()

	// generate request, here we'll use login as example
	reqBody, err := json.Marshal(map[string]string{
		"username": "test",
		"password": "pass",
	})
	require.NoError(t, err)

	client := newHttpTestClient()

	// make request
	resp, err := client.Post(testServer.URL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	defer resp.Body.Close()

	// check for response, here we'll just check for 204 NoContent
	assert.Equalf(t, http.StatusNoContent, resp.StatusCode, "login handler: unexpected http status")

	assert.NotEmpty(t, resp.Header.Get("Set-Cookie"), "handler returned no cookie")

	// validate token
	resp, err = client.Get(testServer.URL + "/auth/validate")
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equalf(t, http.StatusOK, resp.StatusCode, "validate handler: unexpected http status")

	// logout
	resp, err = client.Post(testServer.URL+"/auth/logout", "application/json", nil)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equalf(t, http.StatusNoContent, resp.StatusCode, "logout handler: unexpected http status")

	//if v := resp.Header.Get("Set-Cookie"); v != "" {
	//	t.Errorf("logout handler returned cookie")
	//}
}

func newSessionToken(t *testing.T, sm *scs.SessionManager) string {
	t.Helper()

	ctx, err := sm.Load(context.Background(), "")
	assert.NoError(t, err)

	sm.Put(ctx, "authenticated", true)

	token, _, err := sm.Commit(ctx)
	assert.NoError(t, err)

	return token
}

func TestRevokeOtherSessions_KeepsCurrentDestroysRest(t *testing.T) {
	sm := scs.New()

	current := newSessionToken(t, sm)
	other1 := newSessionToken(t, sm)
	other2 := newSessionToken(t, sm)

	h := &authHandler{log: zerolog.Nop(), sessionManager: sm}

	ctx, err := sm.Load(context.Background(), current)
	assert.NoError(t, err)

	h.revokeOtherSessions(ctx)

	_, found, err := sm.Store.Find(current)
	assert.NoError(t, err)
	assert.True(t, found)

	for _, token := range []string{other1, other2} {
		_, found, err := sm.Store.Find(token)
		assert.NoError(t, err)
		assert.False(t, found)
	}
}

type failingDeleteStore struct {
	*memstore.MemStore
	failToken string
}

func (s *failingDeleteStore) Delete(token string) error {
	if token == s.failToken {
		return errors.New("store unavailable")
	}

	return s.MemStore.Delete(token)
}

func TestRevokeOtherSessions_ContinuesAfterDestroyFailure(t *testing.T) {
	store := &failingDeleteStore{MemStore: memstore.New()}
	sm := scs.New()
	sm.Store = store

	current := newSessionToken(t, sm)
	failing := newSessionToken(t, sm)
	others := []string{newSessionToken(t, sm), newSessionToken(t, sm), newSessionToken(t, sm)}
	store.failToken = failing

	h := &authHandler{log: zerolog.Nop(), sessionManager: sm}

	ctx, err := sm.Load(context.Background(), current)
	require.NoError(t, err)

	h.revokeOtherSessions(ctx)

	for _, token := range []string{current, failing} {
		_, found, err := sm.Store.Find(token)
		assert.NoError(t, err)
		assert.True(t, found)
	}

	for _, token := range others {
		_, found, err := sm.Store.Find(token)
		assert.NoError(t, err)
		assert.False(t, found)
	}
}

func TestAuthHandlerLoginRateLimitsFailures(t *testing.T) {
	srv := NewServer(Deps{
		Log:            zerolog.Nop(),
		Config:         &config.AppConfig{Config: &domain.Config{BaseURL: "/"}},
		SessionManager: scs.New(),
	})
	service := authServiceMock{users: map[string]*domain.User{
		"test": {Username: "test", Password: "pass"},
	}}
	handler := newAuthHandler(encoder{}, zerolog.Nop(), srv, srv.config.Config, srv.sessionManager, service, &oidcAuthServiceMock{})
	router := setupServer(srv)
	router.Route("/auth", handler.Routes)

	login := func(remoteAddr, password string) *httptest.ResponseRecorder {
		t.Helper()

		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"test","password":"`+password+`"}`))
		req.RemoteAddr = remoteAddr
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		return rec
	}

	for range loginFailureBurst - 1 {
		assert.Equal(t, http.StatusForbidden, login("192.0.2.1:1000", "wrong").Code)
	}
	assert.Equal(t, http.StatusNoContent, login("192.0.2.1:1001", "pass").Code, "success resets the failure count")

	for range loginFailureBurst {
		assert.Equal(t, http.StatusForbidden, login("192.0.2.1:1002", "wrong").Code)
	}

	rec := login("192.0.2.1:1003", "pass")
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "correct password is refused while limited")
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))

	assert.Equal(t, http.StatusNoContent, login("192.0.2.2:1000", "pass").Code, "other clients can still log in")
}

func TestAuthHandlerLoginRateLimitIgnoresSpoofedClientHeaders(t *testing.T) {
	const (
		attacker = "203.0.113.9:8888"
		victim   = "198.51.100.7:9999"
		proxy    = "10.0.0.1:7777"
	)

	newHandler := func(t *testing.T, trustedProxies []string) http.Handler {
		t.Helper()

		srv := NewServer(Deps{
			Log:            zerolog.Nop(),
			Config:         &config.AppConfig{Config: &domain.Config{BaseURL: "/", BaseURLModeLegacy: true, TrustedProxies: trustedProxies}},
			SessionManager: scs.New(),
			AuthService:    authServiceMock{users: map[string]*domain.User{"test": {Username: "test", Password: "pass"}}},
			OIDCService:    &oidcAuthServiceMock{},
		})

		return srv.Handler()
	}

	login := func(t *testing.T, handler http.Handler, remoteAddr, password string, headers map[string]string) *httptest.ResponseRecorder {
		t.Helper()

		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"username":"test","password":"`+password+`"}`))
		req.RemoteAddr = remoteAddr
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		return rec
	}

	for _, header := range []string{"X-Real-IP", "True-Client-IP", "X-Forwarded-For"} {
		t.Run("legacy RealIP ignores "+header, func(t *testing.T) {
			handler := newHandler(t, nil)
			spoofed := map[string]string{header: "198.51.100.7"}

			for range loginFailureBurst {
				assert.Equal(t, http.StatusForbidden, login(t, handler, attacker, "wrong", spoofed).Code)
			}

			assert.Equal(t, http.StatusTooManyRequests, login(t, handler, attacker, "pass", spoofed).Code, "the attacker's own peer address is limited")
			assert.Equal(t, http.StatusNoContent, login(t, handler, victim, "pass", nil).Code, "the spoofed address can still log in")
		})

		t.Run("untrusted peer cannot spoof "+header+" past trusted proxies", func(t *testing.T) {
			handler := newHandler(t, []string{"10.0.0.1"})
			spoofed := map[string]string{header: "198.51.100.7"}

			for range loginFailureBurst {
				assert.Equal(t, http.StatusForbidden, login(t, handler, attacker, "wrong", spoofed).Code)
			}

			assert.Equal(t, http.StatusTooManyRequests, login(t, handler, attacker, "pass", spoofed).Code)
			assert.Equal(t, http.StatusNoContent, login(t, handler, victim, "pass", nil).Code)
		})
	}

	t.Run("clients behind a trusted proxy are limited independently", func(t *testing.T) {
		handler := newHandler(t, []string{"10.0.0.1"})

		for range loginFailureBurst {
			assert.Equal(t, http.StatusForbidden, login(t, handler, proxy, "wrong", map[string]string{"X-Forwarded-For": "198.51.100.7"}).Code)
		}

		assert.Equal(t, http.StatusTooManyRequests, login(t, handler, proxy, "pass", map[string]string{"X-Forwarded-For": "198.51.100.7"}).Code)
		assert.Equal(t, http.StatusNoContent, login(t, handler, proxy, "pass", map[string]string{"X-Forwarded-For": "198.51.100.8"}).Code, "a neighbour behind the same proxy is unaffected")
	})
}

func TestAuthHandlerLoginOperationalErrorsDoNotCountAsFailures(t *testing.T) {
	srv := NewServer(Deps{
		Log:            zerolog.Nop(),
		Config:         &config.AppConfig{Config: &domain.Config{BaseURL: "/"}},
		SessionManager: scs.New(),
	})
	service := authServiceMock{loginErr: errors.New("database is locked")}
	handler := newAuthHandler(encoder{}, zerolog.Nop(), srv, srv.config.Config, srv.sessionManager, service, &oidcAuthServiceMock{})
	router := setupServer(srv)
	router.Route("/auth", handler.Routes)

	for range loginFailureBurst + 1 {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"username":"test","password":"pass"}`))
		req.RemoteAddr = "192.0.2.1:1000"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	}
}

func TestAuthHandlerLoginStalledBodyDoesNotHoldThrottle(t *testing.T) {
	srv := NewServer(Deps{
		Log:            zerolog.Nop(),
		Config:         &config.AppConfig{Config: &domain.Config{BaseURL: "/"}},
		SessionManager: scs.New(),
	})
	service := authServiceMock{users: map[string]*domain.User{
		"test": {Username: "test", Password: "pass"},
	}}
	handler := newAuthHandler(encoder{}, zerolog.Nop(), srv, srv.config.Config, srv.sessionManager, service, &oidcAuthServiceMock{})
	router := setupServer(srv)
	router.Route("/auth", handler.Routes)

	testServer := runTestServer(router)
	defer testServer.Close()

	conn, err := net.Dial("tcp", testServer.Listener.Addr().String())
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Write([]byte("POST /auth/login HTTP/1.1\r\nHost: autobrr\r\nContent-Type: application/json\r\nContent-Length: 100\r\n\r\n{"))
	require.NoError(t, err)

	// Give the server time to route the stalled request into the handler chain.
	time.Sleep(100 * time.Millisecond)

	client := newHttpTestClient()
	client.Timeout = 3 * time.Second

	resp, err := client.Post(testServer.URL+"/auth/login", "application/json", strings.NewReader(`{"username":"test","password":"pass"}`))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestAuthHandlerLoginBodyTooLarge(t *testing.T) {
	srv := NewServer(Deps{
		Log:            zerolog.Nop(),
		Config:         &config.AppConfig{Config: &domain.Config{BaseURL: "/"}},
		SessionManager: scs.New(),
	})
	service := authServiceMock{users: map[string]*domain.User{
		"test": {Username: "test", Password: "pass"},
	}}
	handler := newAuthHandler(encoder{}, zerolog.Nop(), srv, srv.config.Config, srv.sessionManager, service, &oidcAuthServiceMock{})
	router := setupServer(srv)
	router.Route("/auth", handler.Routes)

	body := `{"username":"test","password":"pass","padding":"` + strings.Repeat("a", loginBodyLimit) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

type deadlineRecordingOIDCMock struct {
	*oidcAuthServiceMock
	deadline time.Time
}

func (o *deadlineRecordingOIDCMock) OAuthExchange(ctx context.Context, code string, opts ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	o.deadline, _ = ctx.Deadline()

	return nil, errors.New("provider unavailable")
}

func TestOIDCHandlerCallbackBoundsProviderCalls(t *testing.T) {
	sm := scs.New()
	service := &deadlineRecordingOIDCMock{oidcAuthServiceMock: &oidcAuthServiceMock{}}
	h := NewOIDCHandler(encoder{}, zerolog.Nop(), &domain.Config{}, sm, service, func(r *http.Request) string { return "" })

	h.pendingLogins.Set("state", pendingOIDCLogin{}, ttlcache.DefaultTTL)

	req := httptest.NewRequest(http.MethodGet, "/callback?state=state&code=code", nil)
	req.AddCookie(&http.Cookie{Name: stateCookieName("state"), Value: "state"})
	rec := httptest.NewRecorder()
	h.handleCallback(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	require.False(t, service.deadline.IsZero(), "provider call must carry a deadline")
	assert.WithinDuration(t, time.Now().Add(oidcProviderTimeout), service.deadline, 5*time.Second)
}

func TestOIDCHandlerProviderUnavailable(t *testing.T) {
	sm := scs.New()
	h := NewOIDCHandler(encoder{}, zerolog.Nop(), &domain.Config{OIDCEnabled: true, OIDCIssuer: "https://auth.example.com"}, sm, &oidcAuthServiceMock{unavailable: true}, func(r *http.Request) string { return "" })

	rec := httptest.NewRecorder()
	sm.LoadAndSave(http.HandlerFunc(h.getConfig)).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/config", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	var config GetConfigResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &config))
	assert.True(t, config.Enabled)
	assert.Empty(t, config.AuthorizationURL)
	assert.Empty(t, config.State)
	assert.Equal(t, "https://auth.example.com", config.IssuerURL)
	assert.Empty(t, h.pendingLogins.GetKeys())
	assert.Empty(t, rec.Result().Cookies())

	rec = httptest.NewRecorder()
	h.handleCallback(rec, httptest.NewRequest(http.MethodGet, "/callback?state=state&code=code", nil))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
