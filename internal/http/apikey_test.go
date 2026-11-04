// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type apikeyServiceStub struct {
	apikeyService
	tokens  map[string]*domain.APIKey
	ids     map[int]bool
	stored  *domain.APIKey
	updated *domain.APIKey
}

func (s *apikeyServiceStub) List(context.Context) ([]domain.APIKey, error) { return nil, nil }

func (s *apikeyServiceStub) ValidateAPIKey(_ context.Context, token string) (*domain.APIKey, bool) {
	key, ok := s.tokens[token]
	return key, ok
}

func (s *apikeyServiceStub) Store(_ context.Context, key *domain.APIKey) error {
	if err := key.Validate(); err != nil {
		return err
	}

	key.ID = 1
	key.Key = "generated-key"
	key.KeyHash = domain.HashAPIKey(key.Key)
	s.stored = key

	return nil
}

func (s *apikeyServiceStub) Update(_ context.Context, key *domain.APIKey) error {
	if err := key.Validate(); err != nil {
		return err
	}

	if !s.ids[key.ID] {
		return domain.ErrRecordNotFound
	}

	key.KeyHash = domain.HashAPIKey("existing-key")
	s.updated = key

	return nil
}

func newScopedTestServer() *Server {
	service := &apikeyServiceStub{tokens: map[string]*domain.APIKey{
		"full":               {Scopes: []string{domain.APIScopeAll}},
		"config-read":        {Scopes: []string{"config:read"}},
		"filters-read":       {Scopes: []string{"filters:read"}},
		"filters-write":      {Scopes: []string{"filters:write"}},
		"irc-read":           {Scopes: []string{"irc:read"}},
		"logs-read":          {Scopes: []string{"logs:read"}},
		"notifications-read": {Scopes: []string{"notifications:read"}},
		"webhooks-write":     {Scopes: []string{"webhooks:write"}},
	}}

	return &Server{apiService: service, config: &config.AppConfig{Config: &domain.Config{}}}
}

func serveWithToken(router http.Handler, method, target, token, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if token != "" {
		request.Header.Set("X-API-Token", token)
	}

	router.ServeHTTP(recorder, request)

	return recorder
}

func TestAPIKeyScopeMiddleware(t *testing.T) {
	s := newScopedTestServer()

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
		r.With(requireStreamAccess).Get("/events", ok)
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
		{name: "events_logs", method: http.MethodGet, target: "/events?stream=logs", token: "logs-read", want: http.StatusOK},
		{name: "events_logs_cannot_read_notifications", method: http.MethodGet, target: "/events?stream=notifications", token: "logs-read", want: http.StatusForbidden},
		{name: "events_notifications", method: http.MethodGet, target: "/events?stream=notifications", token: "notifications-read", want: http.StatusOK},
		{name: "events_notifications_cannot_read_logs", method: http.MethodGet, target: "/events?stream=logs", token: "notifications-read", want: http.StatusForbidden},
		{name: "events_irc", method: http.MethodGet, target: "/events?stream=irc", token: "irc-read", want: http.StatusOK},
		{name: "events_unknown_stream", method: http.MethodGet, target: "/events?stream=mock", token: "full", want: http.StatusBadRequest},
		{name: "events_full_access", method: http.MethodGet, target: "/events?stream=logs", token: "full", want: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := serveWithToken(router, tt.method, tt.target, tt.token, "")

			assert.Equal(t, tt.want, recorder.Code)
		})
	}
}

func TestIrcEventsOnlyServeIrcStream(t *testing.T) {
	s := newScopedTestServer()

	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(s.IsAuthenticated)
		r.With(requireScope(domain.APIResourceIRC)).Route("/irc", newIrcHandler(encoder{}, nil, nil).Routes)
	})

	for _, stream := range []string{"logs", "notifications", ""} {
		t.Run(stream, func(t *testing.T) {
			recorder := serveWithToken(router, http.MethodGet, "/irc/events?stream="+stream, "irc-read", "")

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}

type commandServiceStub struct {
	filterService
	called bool
}

func (s *commandServiceStub) Store(context.Context, *domain.Filter) error {
	s.called = true
	return nil
}

func (s *commandServiceStub) Update(context.Context, *domain.Filter) error {
	s.called = true
	return nil
}

func (s *commandServiceStub) UpdatePartial(context.Context, domain.FilterUpdate) error {
	s.called = true
	return nil
}

func (s *commandServiceStub) TestExternal(context.Context, *domain.FilterExternal) (*domain.FilterExternalTestResult, error) {
	s.called = true
	return &domain.FilterExternalTestResult{Success: true}, nil
}

type actionCommandServiceStub struct {
	actionService
	called bool
}

func (s *actionCommandServiceStub) Store(context.Context, *domain.Action) error {
	s.called = true
	return nil
}

func (s *actionCommandServiceStub) Update(_ context.Context, action *domain.Action) (*domain.Action, error) {
	s.called = true
	return action, nil
}

func TestCommandsRequireFullAccess(t *testing.T) {
	s := newScopedTestServer()
	filters := &commandServiceStub{}
	actions := &actionCommandServiceStub{}

	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(s.IsAuthenticated)

		r.With(requireScope(domain.APIResourceFilters)).Route("/actions", newActionHandler(encoder{}, actions).Routes)
		r.With(requireScope(domain.APIResourceFilters)).Route("/filters", newFilterHandler(encoder{}, filters).Routes)
	})

	const (
		execFilter   = `{"name":"mock","external":[{"name":"mock","type":"EXEC","exec_cmd":"/bin/sh"}]}`
		execAction   = `{"name":"mock","actions":[{"name":"mock","type":"EXEC","exec_cmd":"/bin/sh"}]}`
		webhookOnly  = `{"name":"mock","external":[{"name":"mock","type":"WEBHOOK"}],"actions":[{"name":"mock","type":"QBITTORRENT"}]}`
		execExternal = `{"name":"mock","type":"EXEC","exec_cmd":"/bin/sh"}`
	)

	tests := []struct {
		name   string
		method string
		target string
		body   string
		token  string
		want   int
	}{
		{name: "store_exec_filter", method: http.MethodPost, target: "/filters/", body: execFilter, token: "filters-write", want: http.StatusForbidden},
		{name: "store_exec_action_in_filter", method: http.MethodPost, target: "/filters/", body: execAction, token: "filters-write", want: http.StatusForbidden},
		{name: "store_without_commands", method: http.MethodPost, target: "/filters/", body: webhookOnly, token: "filters-write", want: http.StatusCreated},
		{name: "store_exec_full_access", method: http.MethodPost, target: "/filters/", body: execFilter, token: "full", want: http.StatusCreated},
		{name: "update_exec_filter", method: http.MethodPut, target: "/filters/1", body: execFilter, token: "filters-write", want: http.StatusForbidden},
		{name: "update_partial_exec", method: http.MethodPatch, target: "/filters/1", body: execAction, token: "filters-write", want: http.StatusForbidden},
		{name: "update_partial_without_commands", method: http.MethodPatch, target: "/filters/1", body: `{"enabled":true}`, token: "filters-write", want: http.StatusNoContent},
		{name: "test_exec_external", method: http.MethodPost, target: "/filters/external/test", body: execExternal, token: "filters-write", want: http.StatusForbidden},
		{name: "test_exec_external_full_access", method: http.MethodPost, target: "/filters/external/test", body: execExternal, token: "full", want: http.StatusOK},
		{name: "store_exec_action", method: http.MethodPost, target: "/actions/", body: `{"name":"mock","type":"EXEC","exec_cmd":"/bin/sh"}`, token: "filters-write", want: http.StatusForbidden},
		{name: "update_exec_action", method: http.MethodPut, target: "/actions/1", body: `{"name":"mock","type":"EXEC","exec_cmd":"/bin/sh"}`, token: "filters-write", want: http.StatusForbidden},
		{name: "store_other_action", method: http.MethodPost, target: "/actions/", body: `{"name":"mock","type":"QBITTORRENT"}`, token: "filters-write", want: http.StatusCreated},
		{name: "store_exec_action_full_access", method: http.MethodPost, target: "/actions/", body: `{"name":"mock","type":"EXEC","exec_cmd":"/bin/sh"}`, token: "full", want: http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters.called = false
			actions.called = false

			recorder := serveWithToken(router, tt.method, tt.target, tt.token, tt.body)

			assert.Equal(t, tt.want, recorder.Code)
			assert.Equal(t, tt.want != http.StatusForbidden, filters.called || actions.called)
		})
	}
}

func TestConfigWriteRequiresFullAccess(t *testing.T) {
	s := newScopedTestServer()

	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(s.IsAuthenticated)
		r.With(requireScope(domain.APIResourceConfig)).Route("/config", newConfigHandler(encoder{}, buildInfo{}, s.config).Routes)
	})

	recorder := serveWithToken(router, http.MethodPatch, "/config/", "config-read", `{"log_level":"DEBUG"}`)

	assert.Equal(t, http.StatusForbidden, recorder.Code)
	assert.Empty(t, s.config.Config.LogLevel)
}

func newAPIKeyTestRouter(cfg *domain.Config) chi.Router {
	h := newAPIKeyHandler(newEncoder(zerolog.Nop()), &apikeyServiceStub{}, cfg)
	r := chi.NewRouter()
	r.Route("/keys", h.Routes)
	return r
}

func TestAPIKeyHandler_BlockedWhenAuthDisabled(t *testing.T) {
	cfg := &domain.Config{
		AuthDisabled:                true,
		AuthDisabledAcknowledgement: domain.AuthDisabledAcknowledgementValue,
	}
	r := newAPIKeyTestRouter(cfg)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/keys/"},
		{http.MethodPost, "/keys/"},
		{http.MethodDelete, "/keys/1"},
	}

	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			assert.Equal(t, http.StatusForbidden, rr.Code)
		})
	}
}

func TestAPIKeyHandlerStore(t *testing.T) {
	service := &apikeyServiceStub{}
	router := chi.NewRouter()
	newAPIKeyHandler(encoder{}, service, &domain.Config{}).Routes(router)

	t.Run("returns the key once", func(t *testing.T) {
		recorder := serveWithToken(router, http.MethodPost, "/", "", `{"name":"sonarr","scopes":["*"]}`)

		assert.Equal(t, http.StatusCreated, recorder.Code)
		assert.Contains(t, recorder.Body.String(), `"key":"generated-key"`)
		assert.Contains(t, recorder.Body.String(), `"id":1`)
		assert.NotContains(t, recorder.Body.String(), service.stored.KeyHash)
	})

	t.Run("rejects null body", func(t *testing.T) {
		recorder := serveWithToken(router, http.MethodPost, "/", "", `null`)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestAPIKeyHandlerUpdate(t *testing.T) {
	service := &apikeyServiceStub{ids: map[int]bool{1: true}}
	router := chi.NewRouter()
	newAPIKeyHandler(encoder{}, service, &domain.Config{}).Routes(router)

	tests := []struct {
		name   string
		target string
		body   string
		want   int
	}{
		{name: "ok", target: "/1", body: `{"id":9,"name":"sonarr","key":"ignored","scopes":["webhooks:write"]}`, want: http.StatusOK},
		{name: "invalid_scopes", target: "/1", body: `{"name":"sonarr","scopes":["logs:write"]}`, want: http.StatusBadRequest},
		{name: "not_found", target: "/2", body: `{"name":"sonarr","scopes":["*"]}`, want: http.StatusNotFound},
		{name: "invalid_id", target: "/mock-key", body: `{"name":"sonarr","scopes":["*"]}`, want: http.StatusBadRequest},
		{name: "null_body", target: "/1", body: `null`, want: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := serveWithToken(router, http.MethodPut, tt.target, "", tt.body)

			assert.Equal(t, tt.want, recorder.Code)
			assert.NotContains(t, recorder.Body.String(), `"key":`)
		})
	}

	assert.Equal(t, 1, service.updated.ID)
	assert.Empty(t, service.updated.Key)
	assert.Equal(t, []string{"webhooks:write"}, service.updated.Scopes)
}

func TestAPIKeyHandler_AllowedWhenAuthEnabled(t *testing.T) {
	r := newAPIKeyTestRouter(&domain.Config{})

	req := httptest.NewRequest(http.MethodGet, "/keys/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
