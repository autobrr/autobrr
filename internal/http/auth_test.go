// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type authServiceMock struct {
	users map[string]*domain.User
}

func (a authServiceMock) GetUserCount(ctx context.Context) (int, error) {
	return len(a.users), nil
}

func (a authServiceMock) Login(ctx context.Context, username, password string) (*domain.User, error) {
	u, ok := a.users[username]
	if !ok {
		return nil, errors.New("invalid login")
	}

	if u.Password != password {
		return nil, errors.New("bad credentials")
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
}

func (o *oidcAuthServiceMock) IsEnabled() bool {
	return false
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
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	client := newHttpTestClient()

	// make request
	resp, err := client.Post(testServer.URL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	defer resp.Body.Close()

	// check for response, here we'll just check for 204 NoContent
	assert.Equalf(t, http.StatusNoContent, resp.StatusCode, "login handler: unexpected http status")

	if v := resp.Header.Get("Set-Cookie"); v == "" {
		t.Errorf("handler returned no cookie")
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
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	client := newHttpTestClient()

	// make request
	resp, err := client.Post(testServer.URL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	defer resp.Body.Close()

	// check for response, here we'll just check for 204 NoContent
	assert.Equalf(t, http.StatusNoContent, resp.StatusCode, "login handler: bad response")

	if v := resp.Header.Get("Set-Cookie"); v == "" {
		assert.Equalf(t, "", v, "login handler: expected Set-Cookie header")
	}

	// validate token
	resp, err = client.Get(testServer.URL + "/auth/validate")
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	defer resp.Body.Close()

	assert.Equalf(t, http.StatusOK, resp.StatusCode, "validate handler: unexpected http status")
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
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

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
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	client := newHttpTestClient()

	// make request
	resp, err := client.Post(testServer.URL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

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
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	client := newHttpTestClient()

	// make request
	resp, err := client.Post(testServer.URL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	defer resp.Body.Close()

	// check for response, here we'll just check for 204 NoContent
	if status := resp.StatusCode; status != http.StatusNoContent {
		t.Errorf("login: handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}

	assert.Equalf(t, http.StatusNoContent, resp.StatusCode, "login handler: unexpected http status")

	if v := resp.Header.Get("Set-Cookie"); v == "" {
		t.Errorf("handler returned no cookie")
	}

	// validate token
	resp, err = client.Get(testServer.URL + "/auth/validate")
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	defer resp.Body.Close()

	assert.Equalf(t, http.StatusOK, resp.StatusCode, "validate handler: unexpected http status")

	// logout
	resp, err = client.Post(testServer.URL+"/auth/logout", "application/json", nil)
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	defer resp.Body.Close()

	assert.Equalf(t, http.StatusNoContent, resp.StatusCode, "logout handler: unexpected http status")

	//if v := resp.Header.Get("Set-Cookie"); v != "" {
	//	t.Errorf("logout handler returned cookie")
	//}
}
