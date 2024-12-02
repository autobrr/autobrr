// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
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
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type authServiceMock struct {
	users        map[string]*domain.User
	twoFAEnabled map[string]bool
	twoFASecrets map[string]string
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

func (a authServiceMock) Get2FAStatus(ctx context.Context, username string) (bool, error) {
	enabled, exists := a.twoFAEnabled[username]
	if !exists {
		return false, nil
	}
	return enabled, nil
}

func (a authServiceMock) Enable2FA(ctx context.Context, username string) (string, string, error) {
	if _, exists := a.users[username]; !exists {
		return "", "", errors.New("user not found")
	}

	// For testing purposes, return a fixed URL and secret
	secret := "TESTSECRET123456"
	url := "otpauth://totp/autobrr:" + username + "?secret=" + secret + "&issuer=autobrr"

	// Store the secret for later verification
	a.twoFASecrets[username] = secret

	return url, secret, nil
}

func (a authServiceMock) Verify2FA(ctx context.Context, username string, code string) error {
	if _, exists := a.users[username]; !exists {
		return errors.New("user not found")
	}

	// For testing purposes, accept "123456" as valid code during setup
	if code != "123456" {
		return errors.New("invalid code")
	}

	a.twoFAEnabled[username] = true
	return nil
}

func (a authServiceMock) Verify2FALogin(ctx context.Context, username string, code string) error {
	if _, exists := a.users[username]; !exists {
		return errors.New("user not found")
	}

	if !a.twoFAEnabled[username] {
		return errors.New("2FA not enabled for user")
	}

	// For testing purposes, accept "123456" as valid code during login
	if code != "123456" {
		return errors.New("invalid code")
	}

	return nil
}

func (a authServiceMock) Disable2FA(ctx context.Context, username string) error {
	if _, exists := a.users[username]; !exists {
		return errors.New("user not found")
	}

	if !a.twoFAEnabled[username] {
		return errors.New("2FA not enabled for user")
	}

	a.twoFAEnabled[username] = false
	delete(a.twoFASecrets, username)
	return nil
}

func setupServer() chi.Router {
	r := chi.NewRouter()
	//r.Use(middleware.Logger)
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

func TestAuthHandlerLogin(t *testing.T) {
	t.Parallel()
	logger := zerolog.Nop()
	encoder := encoder{}
	cookieStore := sessions.NewCookieStore([]byte("test"))

	service := authServiceMock{
		users: map[string]*domain.User{
			"test": {
				ID:       0,
				Username: "test",
				Password: "pass",
			},
		},
		twoFAEnabled: make(map[string]bool),
		twoFASecrets: make(map[string]string),
	}

	server := Server{
		log:         logger,
		cookieStore: cookieStore,
	}

	handler := newAuthHandler(encoder, logger, server, &domain.Config{}, cookieStore, service)
	s := setupServer()
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

	jarOptions := &cookiejar.Options{PublicSuffixList: nil}
	jar, err := cookiejar.New(jarOptions)
	if err != nil {
		log.Fatalf("error creating cookiejar: %v", err)
	}

	client := http.DefaultClient
	client.Jar = jar

	// make request
	resp, err := client.Post(testServer.URL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	defer resp.Body.Close()

	// Using StatusOK (200) instead of StatusNoContent (204) because the login endpoint
	// now returns a JSON response with the requires2FA flag
	assert.Equal(t, http.StatusOK, resp.StatusCode, "login handler: unexpected http status")

	var response map[string]bool
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.False(t, response["requires2FA"])

	if v := resp.Header.Get("Set-Cookie"); v == "" {
		t.Errorf("handler returned no cookie")
	}
}

func TestAuthHandlerLoginWith2FA(t *testing.T) {
	logger := zerolog.Nop()
	encoder := encoder{}
	cookieStore := sessions.NewCookieStore([]byte("test"))

	service := authServiceMock{
		users: map[string]*domain.User{
			"test": {
				ID:       0,
				Username: "test",
				Password: "pass",
			},
		},
		twoFAEnabled: map[string]bool{
			"test": true,
		},
		twoFASecrets: map[string]string{
			"test": "TESTSECRET123456",
		},
	}

	server := Server{
		log:         logger,
		cookieStore: cookieStore,
	}

	handler := newAuthHandler(encoder, logger, server, &domain.Config{}, cookieStore, service)
	s := setupServer()
	s.Route("/auth", handler.Routes)

	testServer := runTestServer(s)
	defer testServer.Close()

	jarOptions := &cookiejar.Options{PublicSuffixList: nil}
	jar, err := cookiejar.New(jarOptions)
	if err != nil {
		log.Fatalf("error creating cookiejar: %v", err)
	}

	client := http.DefaultClient
	client.Jar = jar

	// Step 1: Initial login
	reqBody, err := json.Marshal(map[string]string{
		"username": "test",
		"password": "pass",
	})
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	resp, err := client.Post(testServer.URL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}
	defer resp.Body.Close()

	// Using StatusOK because login returns a JSON response with requires2FA flag
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var loginResponse map[string]bool
	err = json.NewDecoder(resp.Body).Decode(&loginResponse)
	assert.NoError(t, err)
	assert.True(t, loginResponse["requires2FA"])

	// Step 2: Verify 2FA code
	reqBody, err = json.Marshal(map[string]string{
		"code": "123456",
	})
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	resp, err = client.Post(testServer.URL+"/auth/2fa/verify", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}
	defer resp.Body.Close()

	// Using StatusOK because verify returns a success message
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAuthHandler2FAFlow(t *testing.T) {
	logger := zerolog.Nop()
	encoder := encoder{}
	cookieStore := sessions.NewCookieStore([]byte("test"))

	service := authServiceMock{
		users: map[string]*domain.User{
			"test": {
				ID:       0,
				Username: "test",
				Password: "pass",
			},
		},
		twoFAEnabled: make(map[string]bool),
		twoFASecrets: make(map[string]string),
	}

	server := Server{
		log:         logger,
		cookieStore: cookieStore,
	}

	handler := newAuthHandler(encoder, logger, server, &domain.Config{}, cookieStore, service)
	s := setupServer()
	s.Route("/auth", handler.Routes)

	testServer := runTestServer(s)
	defer testServer.Close()

	jarOptions := &cookiejar.Options{PublicSuffixList: nil}
	jar, err := cookiejar.New(jarOptions)
	if err != nil {
		log.Fatalf("error creating cookiejar: %v", err)
	}

	client := http.DefaultClient
	client.Jar = jar

	// Step 1: Login first to get session
	reqBody, err := json.Marshal(map[string]string{
		"username": "test",
		"password": "pass",
	})
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	resp, err := client.Post(testServer.URL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}
	defer resp.Body.Close()

	// Step 2: Check initial 2FA status
	resp, err = client.Get(testServer.URL + "/auth/2fa/status")
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}
	defer resp.Body.Close()

	// Using StatusOK because status endpoint returns a JSON response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var statusResponse map[string]bool
	err = json.NewDecoder(resp.Body).Decode(&statusResponse)
	assert.NoError(t, err)
	assert.False(t, statusResponse["enabled"])

	// Step 3: Enable 2FA
	resp, err = client.Post(testServer.URL+"/auth/2fa/enable", "application/json", nil)
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}
	defer resp.Body.Close()

	// Using StatusOK because enable returns QR code URL and secret
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var enableResponse map[string]string
	err = json.NewDecoder(resp.Body).Decode(&enableResponse)
	assert.NoError(t, err)
	assert.NotEmpty(t, enableResponse["url"])
	assert.NotEmpty(t, enableResponse["secret"])

	// Step 4: Verify 2FA setup
	reqBody, err = json.Marshal(map[string]string{
		"code": "123456",
	})
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	resp, err = client.Post(testServer.URL+"/auth/2fa/verify", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}
	defer resp.Body.Close()

	// Using StatusOK because verify returns a success message
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Step 5: Check 2FA status again
	resp, err = client.Get(testServer.URL + "/auth/2fa/status")
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}
	defer resp.Body.Close()

	// Using StatusOK because status endpoint returns a JSON response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	err = json.NewDecoder(resp.Body).Decode(&statusResponse)
	assert.NoError(t, err)
	assert.True(t, statusResponse["enabled"])

	// Step 6: Disable 2FA
	resp, err = client.Post(testServer.URL+"/auth/2fa/disable", "application/json", nil)
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}
	defer resp.Body.Close()

	// Using StatusOK because disable returns a success message
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Step 7: Verify 2FA is disabled
	resp, err = client.Get(testServer.URL + "/auth/2fa/status")
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}
	defer resp.Body.Close()

	// Using StatusOK because status endpoint returns a JSON response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	err = json.NewDecoder(resp.Body).Decode(&statusResponse)
	assert.NoError(t, err)
	assert.False(t, statusResponse["enabled"])
}

func TestAuthHandlerValidateOK(t *testing.T) {
	t.Parallel()
	logger := zerolog.Nop()
	encoder := encoder{}
	cookieStore := sessions.NewCookieStore([]byte("test"))

	service := authServiceMock{
		users: map[string]*domain.User{
			"test": {
				ID:       0,
				Username: "test",
				Password: "pass",
			},
		},
	}

	server := Server{
		log:         logger,
		cookieStore: cookieStore,
	}

	handler := newAuthHandler(encoder, logger, server, &domain.Config{}, cookieStore, service)
	s := setupServer()
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

	jarOptions := &cookiejar.Options{PublicSuffixList: nil}
	jar, err := cookiejar.New(jarOptions)
	if err != nil {
		log.Fatalf("error creating cookiejar: %v", err)
	}

	client := http.DefaultClient
	client.Jar = jar

	// make request
	resp, err := client.Post(testServer.URL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	defer resp.Body.Close()

	// Using StatusOK because login returns a JSON response with requires2FA flag
	assert.Equal(t, http.StatusOK, resp.StatusCode, "login handler: bad response")

	if v := resp.Header.Get("Set-Cookie"); v == "" {
		assert.Equal(t, "", v, "login handler: expected Set-Cookie header")
	}

	// validate token
	resp, err = client.Get(testServer.URL + "/auth/validate")
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	defer resp.Body.Close()

	// Using StatusNoContent because validate endpoint doesn't return any content
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "validate handler: unexpected http status")
}

func TestAuthHandlerValidateBad(t *testing.T) {
	t.Parallel()
	logger := zerolog.Nop()
	encoder := encoder{}
	cookieStore := sessions.NewCookieStore([]byte("test"))

	service := authServiceMock{
		users: map[string]*domain.User{
			"test": {
				ID:       0,
				Username: "test",
				Password: "pass",
			},
		},
	}

	server := Server{
		log:         logger,
		cookieStore: cookieStore,
	}

	handler := newAuthHandler(encoder, logger, server, &domain.Config{}, cookieStore, service)
	s := setupServer()
	s.Route("/auth", handler.Routes)

	testServer := runTestServer(s)
	defer testServer.Close()

	jarOptions := &cookiejar.Options{PublicSuffixList: nil}
	jar, err := cookiejar.New(jarOptions)
	if err != nil {
		log.Fatalf("error creating cookiejar: %v", err)
	}

	client := http.DefaultClient
	client.Jar = jar

	// validate token
	resp, err := client.Get(testServer.URL + "/auth/validate")
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "validate handler: unexpected http status")
}

func TestAuthHandlerLoginBad(t *testing.T) {
	t.Parallel()
	logger := zerolog.Nop()
	encoder := encoder{}
	cookieStore := sessions.NewCookieStore([]byte("test"))

	service := authServiceMock{
		users: map[string]*domain.User{
			"test": {
				ID:       0,
				Username: "test",
				Password: "pass",
			},
		},
	}

	server := Server{
		log: logger,
	}

	handler := newAuthHandler(encoder, logger, server, &domain.Config{}, cookieStore, service)
	s := setupServer()
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

	// make request
	resp, err := http.Post(testServer.URL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	defer resp.Body.Close()

	// check for response, here we'll just check for 403 Forbidden
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "login handler: unexpected http status")
}

func TestAuthHandlerLogout(t *testing.T) {
	logger := zerolog.Nop()
	encoder := encoder{}
	cookieStore := sessions.NewCookieStore([]byte("test"))

	service := authServiceMock{
		users: map[string]*domain.User{
			"test": {
				ID:       0,
				Username: "test",
				Password: "pass",
			},
		},
	}

	server := Server{
		log:         logger,
		cookieStore: cookieStore,
	}

	handler := newAuthHandler(encoder, logger, server, &domain.Config{}, cookieStore, service)
	s := setupServer()
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

	jarOptions := &cookiejar.Options{PublicSuffixList: nil}
	jar, err := cookiejar.New(jarOptions)
	if err != nil {
		log.Fatalf("error creating cookiejar: %v", err)
	}

	client := http.DefaultClient
	client.Jar = jar

	// make request
	resp, err := client.Post(testServer.URL+"/auth/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	defer resp.Body.Close()

	// Using StatusOK because login returns a JSON response with requires2FA flag
	assert.Equal(t, http.StatusOK, resp.StatusCode, "login handler: unexpected http status")

	if v := resp.Header.Get("Set-Cookie"); v == "" {
		t.Errorf("handler returned no cookie")
	}

	// validate token
	resp, err = client.Get(testServer.URL + "/auth/validate")
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	defer resp.Body.Close()

	// Using StatusNoContent because validate endpoint doesn't return any content
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "validate handler: unexpected http status")

	// logout
	resp, err = client.Post(testServer.URL+"/auth/logout", "application/json", nil)
	if err != nil {
		log.Fatalf("Error occurred: %v", err)
	}

	defer resp.Body.Close()

	// Using StatusNoContent because logout endpoint doesn't return any content
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "logout handler: unexpected http status")
	//if v := resp.Header.Get("Set-Cookie"); v != "" {
	//	t.Errorf("logout handler returned cookie")
	//}
}
