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

	assert.Equalf(t, http.StatusNoContent, resp.StatusCode, "validate handler: unexpected http status")
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

	assert.Equalf(t, http.StatusForbidden, resp.StatusCode, "validate handler: unexpected http status")
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
	assert.Equalf(t, http.StatusForbidden, resp.StatusCode, "login handler: unexpected http status")
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

	assert.Equalf(t, http.StatusNoContent, resp.StatusCode, "validate handler: unexpected http status")

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
