// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

type apikeyServiceStub struct {
	apikeyService
	keys    map[string]*domain.APIKey
	updated *domain.APIKey
}

func (s *apikeyServiceStub) ValidateAPIKey(_ context.Context, token string) (*domain.APIKey, bool) {
	key, ok := s.keys[token]
	return key, ok
}

func (s *apikeyServiceStub) Update(_ context.Context, key *domain.APIKey) error {
	if err := key.Validate(); err != nil {
		return err
	}

	if _, ok := s.keys[key.Key]; !ok {
		return domain.ErrRecordNotFound
	}

	s.updated = key

	return nil
}

func TestAPIKeyScopeMiddleware(t *testing.T) {
	service := &apikeyServiceStub{keys: map[string]*domain.APIKey{
		"full":           {Key: "full", Scopes: []string{domain.APIScopeAll}},
		"filters-read":   {Key: "filters-read", Scopes: []string{"filters:read"}},
		"filters-write":  {Key: "filters-write", Scopes: []string{"filters:write"}},
		"webhooks-write": {Key: "webhooks-write", Scopes: []string{"webhooks:write"}},
	}}
	s := &Server{apiService: service}

	ok := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(s.IsAuthenticated)

		r.With(requireScope(domain.APIResourceFilters)).Route("/filters", func(r chi.Router) {
			r.Get("/", ok)
			r.Post("/", ok)
			r.With(requireAccess(domain.APIResourceFilters, domain.APIAccessWrite)).Get("/duplicate", ok)
		})
		r.With(requireAccess(domain.APIResourceWebhooks, domain.APIAccessWrite)).Route("/webhook", func(r chi.Router) {
			r.Get("/trigger", ok)
		})
		r.With(requireFullAccess).Route("/keys", func(r chi.Router) {
			r.Get("/", ok)
		})
	})

	tests := []struct {
		name   string
		method string
		target string
		token  string
		want   int
	}{
		{name: "invalid_key", method: http.MethodGet, target: "/filters/", token: "nope", want: http.StatusUnauthorized},
		{name: "full_access_read", method: http.MethodGet, target: "/filters/", token: "full", want: http.StatusOK},
		{name: "full_access_keys", method: http.MethodGet, target: "/keys/", token: "full", want: http.StatusOK},
		{name: "read_get", method: http.MethodGet, target: "/filters/", token: "filters-read", want: http.StatusOK},
		{name: "read_post", method: http.MethodPost, target: "/filters/", token: "filters-read", want: http.StatusForbidden},
		{name: "read_mutating_get", method: http.MethodGet, target: "/filters/duplicate", token: "filters-read", want: http.StatusForbidden},
		{name: "write_post", method: http.MethodPost, target: "/filters/", token: "filters-write", want: http.StatusOK},
		{name: "write_mutating_get", method: http.MethodGet, target: "/filters/duplicate", token: "filters-write", want: http.StatusOK},
		{name: "scoped_keys", method: http.MethodGet, target: "/keys/", token: "filters-write", want: http.StatusForbidden},
		{name: "webhook_trigger", method: http.MethodGet, target: "/webhook/trigger", token: "webhooks-write", want: http.StatusOK},
		{name: "webhook_other_scope", method: http.MethodGet, target: "/webhook/trigger", token: "filters-write", want: http.StatusForbidden},
		{name: "other_resource", method: http.MethodGet, target: "/filters/", token: "webhooks-write", want: http.StatusForbidden},
		{name: "query_param", method: http.MethodGet, target: "/filters/?apikey=filters-read", want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.target, nil)
			if tt.token != "" {
				request.Header.Set("X-API-Token", tt.token)
			}

			router.ServeHTTP(recorder, request)

			assert.Equal(t, tt.want, recorder.Code)
		})
	}
}

func TestAPIKeyHandlerUpdate(t *testing.T) {
	service := &apikeyServiceStub{keys: map[string]*domain.APIKey{
		"mock-key": {Key: "mock-key", Scopes: []string{domain.APIScopeAll}},
	}}
	router := chi.NewRouter()
	newAPIKeyHandler(encoder{}, service).Routes(router)

	tests := []struct {
		name   string
		target string
		body   string
		want   int
	}{
		{name: "ok", target: "/mock-key", body: `{"name":"sonarr","key":"ignored","scopes":["webhooks:write"]}`, want: http.StatusOK},
		{name: "invalid_scopes", target: "/mock-key", body: `{"name":"sonarr","scopes":["logs:write"]}`, want: http.StatusBadRequest},
		{name: "not_found", target: "/missing", body: `{"name":"sonarr","scopes":["*"]}`, want: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, tt.target, strings.NewReader(tt.body))

			router.ServeHTTP(recorder, request)

			assert.Equal(t, tt.want, recorder.Code)
		})
	}

	assert.Equal(t, "mock-key", service.updated.Key)
	assert.Equal(t, []string{"webhooks:write"}, service.updated.Scopes)
}
