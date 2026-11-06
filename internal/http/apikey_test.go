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

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type oidcServiceStub struct {
	oidcService
}

func (oidcServiceStub) IsEnabled() bool { return false }

// asCaller stands in for IsAuthenticated in tests that mount handlers directly.
func asCaller(caller *Caller) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, withCaller(r, caller))
		})
	}
}

var testUser = &Caller{Method: domain.AuthMethodPassword, Username: "admin"}

func newScopedTestServer() *Server {
	service := &apikeyServiceStub{tokens: map[string]*domain.APIKey{
		"no-access":          {Scopes: []string{"mock:read"}},
		"integrations":       {Scopes: []string{"filters:write", "lists:invoke"}},
		"config-read":        {Scopes: []string{"config:read"}},
		"feeds-invoke":       {Scopes: []string{"feeds:invoke"}},
		"filters-read":       {Scopes: []string{"filters:read"}},
		"filters-write":      {Scopes: []string{"filters:write"}},
		"irc-read":           {Scopes: []string{"irc:read"}},
		"irc-invoke":         {Scopes: []string{"irc:invoke"}},
		"matrix-read":        {Scopes: []string{"matrix:read"}},
		"lists-read":         {Scopes: []string{"lists:read"}},
		"lists-invoke":       {Scopes: []string{"lists:invoke"}},
		"lists-write":        {Scopes: []string{"lists:write"}},
		"logs-read":          {Scopes: []string{"logs:read"}},
		"notifications-read": {Scopes: []string{"notifications:read"}},
		"releases-invoke":    {Scopes: []string{"releases:invoke"}},
	}}

	return &Server{
		apiService:     service,
		oidcService:    oidcServiceStub{},
		sessionManager: scs.New(),
		config:         &config.AppConfig{Config: &domain.Config{}},
	}
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

// apiRouteHandlers returns every route of the real API router keyed by "METHOD /pattern", wrapped in the
// middlewares it is served with.
func apiRouteHandlers(t *testing.T, s *Server) map[string]http.Handler {
	t.Helper()

	handlers := map[string]http.Handler{}
	err := chi.Walk(s.apiRoutes(), func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		handlers[method+" "+route] = chi.Chain(middlewares...).Handler(handler)
		return nil
	})
	require.NoError(t, err)

	return handlers
}

// routeAllows reports whether a request with token gets past the route's scope check. The test server has no
// services, so a handler that runs panics or fails on its input; only the scope checks answer 403.
func routeAllows(handler http.Handler, method, token string) (allowed bool) {
	defer func() {
		if recover() != nil {
			allowed = true
		}
	}()

	return serveWithToken(handler, method, "/?stream=logs", token, "").Code != http.StatusForbidden
}

func TestAPIRoutesRequireScope(t *testing.T) {
	handlers := apiRouteHandlers(t, newScopedTestServer())

	checked := 0
	for route, handler := range handlers {
		method, pattern, _ := strings.Cut(route, " ")
		if strings.HasPrefix(pattern, "/auth/") || strings.HasPrefix(pattern, "/healthz/") {
			continue
		}

		checked++
		assert.False(t, routeAllows(handler, method, "no-access"), "%s has no scope check", route)
	}

	assert.Greater(t, checked, 100)
}

func TestAPIRouteAccess(t *testing.T) {
	handlers := apiRouteHandlers(t, newScopedTestServer())

	tests := []struct {
		route string
		token string
		want  bool
	}{
		{route: "GET /lists/", token: "lists-read", want: true},
		{route: "GET /lists/", token: "lists-invoke", want: false},
		{route: "POST /lists/refresh", token: "lists-invoke", want: true},
		{route: "POST /lists/refresh", token: "lists-read", want: false},
		{route: "POST /lists/refresh", token: "lists-write", want: true},
		{route: "POST /lists/{listID}/refresh", token: "lists-invoke", want: true},
		{route: "PUT /lists/{listID}/", token: "lists-invoke", want: false},
		{route: "PUT /lists/{listID}/", token: "lists-write", want: true},
		{route: "GET /webhook/lists/trigger", token: "lists-invoke", want: true},
		{route: "POST /webhook/lists/trigger/arr", token: "lists-invoke", want: true},
		{route: "POST /webhook/lists/trigger/{listID}", token: "lists-invoke", want: true},
		{route: "POST /webhook/lists/trigger", token: "filters-write", want: false},
		{route: "POST /feeds/{feedID}/forcerun", token: "feeds-invoke", want: true},
		{route: "GET /feeds/", token: "feeds-invoke", want: false},
		{route: "POST /feeds/test", token: "feeds-invoke", want: false},
		{route: "POST /release/{releaseID}/actions/{actionStatusID}/retry", token: "releases-invoke", want: true},
		{route: "POST /release/cleanup-jobs/{jobID}/run", token: "releases-invoke", want: true},
		{route: "DELETE /release/", token: "releases-invoke", want: false},
		{route: "POST /irc/network/{networkID}/restart", token: "irc-invoke", want: true},
		{route: "POST /irc/network/{networkID}/cmd", token: "irc-invoke", want: false},
		{route: "GET /matrix/", token: "matrix-read", want: true},
		{route: "GET /matrix/", token: "irc-read", want: false},
		{route: "GET /matrix/network/{networkID}/room/history", token: "matrix-read", want: true},
		{route: "POST /matrix/network/{networkID}/token", token: "matrix-read", want: false},
		{route: "POST /matrix/network/{networkID}/login", token: "matrix-read", want: false},
		{route: "GET /filters/", token: "filters-read", want: true},
		{route: "POST /filters/", token: "filters-read", want: false},
		{route: "POST /filters/{filterID}/duplicate", token: "filters-read", want: false},
		{route: "POST /filters/{filterID}/duplicate", token: "filters-write", want: true},
		{route: "PUT /actions/{actionID}/", token: "filters-write", want: true},
		{route: "PUT /filters/{filterID}/", token: "integrations", want: true},
		{route: "POST /lists/refresh", token: "integrations", want: true},
		{route: "GET /config/", token: "config-read", want: true},
		{route: "PATCH /config/", token: "config-read", want: false},
		{route: "GET /keys/", token: "integrations", want: false},
		{route: "GET /alerts/", token: "integrations", want: false},
		{route: "GET /events", token: "logs-read", want: true},
		{route: "GET /events", token: "notifications-read", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.route+" "+tt.token, func(t *testing.T) {
			handler, ok := handlers[tt.route]
			require.True(t, ok, "no route %s", tt.route)

			method, _, _ := strings.Cut(tt.route, " ")
			assert.Equal(t, tt.want, routeAllows(handler, method, tt.token))
		})
	}
}

func TestAPIKeyScopeMiddleware(t *testing.T) {
	s := newScopedTestServer()

	ok := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(s.IsAuthenticated)

		r.With(requireScope(domain.APIResourceFilters, domain.APIAccessRead)).Get("/filters", ok)
		r.With(requireStreamAccess).Get("/events", ok)
	})

	tests := []struct {
		name   string
		target string
		token  string
		want   int
	}{
		{name: "invalid_key", target: "/filters", token: "nope", want: http.StatusUnauthorized},
		{name: "query_param", target: "/filters?apikey=filters-read", want: http.StatusOK},
		{name: "events_logs", target: "/events?stream=logs", token: "logs-read", want: http.StatusOK},
		{name: "events_logs_cannot_read_notifications", target: "/events?stream=notifications", token: "logs-read", want: http.StatusForbidden},
		{name: "events_notifications", target: "/events?stream=notifications", token: "notifications-read", want: http.StatusOK},
		{name: "events_notifications_cannot_read_logs", target: "/events?stream=logs", token: "notifications-read", want: http.StatusForbidden},
		{name: "events_irc", target: "/events?stream=irc", token: "irc-read", want: http.StatusOK},
		{name: "events_matrix", target: "/events?stream=matrix", token: "matrix-read", want: http.StatusOK},
		{name: "events_irc_cannot_read_matrix", target: "/events?stream=matrix", token: "irc-read", want: http.StatusForbidden},
		{name: "events_unknown_stream", target: "/events?stream=mock", token: "logs-read", want: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := serveWithToken(router, http.MethodGet, tt.target, tt.token, "")

			assert.Equal(t, tt.want, recorder.Code)
		})
	}
}

func TestCallerMiddlewares(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middlewares := []struct {
		name       string
		middleware func(http.Handler) http.Handler
	}{
		{name: "requireScope", middleware: requireScope(domain.APIResourceLogs, domain.APIAccessRead)},
		{name: "requireUser", middleware: requireUser},
		{name: "requireStreamAccess", middleware: requireStreamAccess},
	}

	callers := []struct {
		name   string
		caller *Caller
		want   map[string]int
	}{
		{name: "no caller", want: map[string]int{"requireScope": http.StatusForbidden, "requireUser": http.StatusForbidden, "requireStreamAccess": http.StatusForbidden}},
		{name: "password", caller: testUser, want: map[string]int{"requireScope": http.StatusOK, "requireUser": http.StatusOK, "requireStreamAccess": http.StatusOK}},
		{name: "oidc", caller: &Caller{Method: domain.AuthMethodOIDC}, want: map[string]int{"requireScope": http.StatusOK, "requireUser": http.StatusOK, "requireStreamAccess": http.StatusOK}},
		{name: "disabled", caller: &Caller{Method: domain.AuthMethodDisabled}, want: map[string]int{"requireScope": http.StatusOK, "requireUser": http.StatusOK, "requireStreamAccess": http.StatusOK}},
		{name: "scoped key", caller: &Caller{Method: domain.AuthMethodAPIKey, APIKey: &domain.APIKey{Scopes: []string{"logs:read"}}}, want: map[string]int{"requireScope": http.StatusOK, "requireUser": http.StatusForbidden, "requireStreamAccess": http.StatusOK}},
		{name: "unscoped key", caller: &Caller{Method: domain.AuthMethodAPIKey, APIKey: &domain.APIKey{Scopes: []string{"filters:read"}}}, want: map[string]int{"requireScope": http.StatusForbidden, "requireUser": http.StatusForbidden, "requireStreamAccess": http.StatusForbidden}},
		{name: "key method without key", caller: &Caller{Method: domain.AuthMethodAPIKey}, want: map[string]int{"requireScope": http.StatusForbidden, "requireUser": http.StatusForbidden, "requireStreamAccess": http.StatusForbidden}},
	}
	for _, c := range callers {
		for _, m := range middlewares {
			t.Run(c.name+" "+m.name, func(t *testing.T) {
				handler := m.middleware(ok)
				if c.caller != nil {
					handler = asCaller(c.caller)(handler)
				}

				assert.Equal(t, c.want[m.name], serveWithToken(handler, http.MethodGet, "/?stream=logs", "", "").Code)
			})
		}
	}
}

func TestIrcEventsOnlyServeIrcStream(t *testing.T) {
	s := newScopedTestServer()

	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(s.IsAuthenticated)
		r.Route("/irc", newIrcHandler(encoder{}, nil, nil).Routes)
	})

	for _, stream := range []string{"logs", "notifications", ""} {
		t.Run(stream, func(t *testing.T) {
			recorder := serveWithToken(router, http.MethodGet, "/irc/events?stream="+stream, "irc-read", "")

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}

func TestMatrixEventsOnlyServeMatrixStream(t *testing.T) {
	s := newScopedTestServer()

	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(s.IsAuthenticated)
		r.Route("/matrix", newMatrixHandler(encoder{}, nil, nil).Routes)
	})

	for _, stream := range []string{"irc", "logs", ""} {
		t.Run(stream, func(t *testing.T) {
			recorder := serveWithToken(router, http.MethodGet, "/matrix/events?stream="+stream, "matrix-read", "")

			assert.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}

func newAPIKeyTestRouter(cfg *domain.Config) chi.Router {
	h := newAPIKeyHandler(newEncoder(zerolog.Nop()), zerolog.Nop(), &apikeyServiceStub{}, cfg)
	r := chi.NewRouter()
	r.Use(asCaller(testUser))
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
	router.Use(asCaller(testUser))
	newAPIKeyHandler(encoder{}, zerolog.Nop(), service, &domain.Config{}).Routes(router)

	t.Run("returns the key once", func(t *testing.T) {
		recorder := serveWithToken(router, http.MethodPost, "/", "", `{"name":"sonarr","scopes":["filters:write"]}`)

		assert.Equal(t, http.StatusCreated, recorder.Code)
		assert.Contains(t, recorder.Body.String(), `"key":"generated-key"`)
		assert.Contains(t, recorder.Body.String(), `"id":1`)
		assert.NotContains(t, recorder.Body.String(), service.stored.KeyHash)
	})

	t.Run("rejects the wildcard scope", func(t *testing.T) {
		recorder := serveWithToken(router, http.MethodPost, "/", "", `{"name":"sonarr","scopes":["*"]}`)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("rejects null body", func(t *testing.T) {
		recorder := serveWithToken(router, http.MethodPost, "/", "", `null`)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestAPIKeyHandlerUpdate(t *testing.T) {
	service := &apikeyServiceStub{ids: map[int]bool{1: true}}
	router := chi.NewRouter()
	router.Use(asCaller(testUser))
	newAPIKeyHandler(encoder{}, zerolog.Nop(), service, &domain.Config{}).Routes(router)

	tests := []struct {
		name   string
		target string
		body   string
		want   int
	}{
		{name: "ok", target: "/1", body: `{"id":9,"name":"sonarr","key":"ignored","scopes":["lists:invoke"]}`, want: http.StatusOK},
		{name: "invalid_scopes", target: "/1", body: `{"name":"sonarr","scopes":["logs:write"]}`, want: http.StatusBadRequest},
		{name: "not_found", target: "/2", body: `{"name":"sonarr","scopes":["filters:write"]}`, want: http.StatusNotFound},
		{name: "invalid_id", target: "/mock-key", body: `{"name":"sonarr","scopes":["filters:write"]}`, want: http.StatusBadRequest},
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
	assert.Equal(t, []string{"lists:invoke"}, service.updated.Scopes)
}

func TestAPIKeyHandler_AllowedWhenAuthEnabled(t *testing.T) {
	r := newAPIKeyTestRouter(&domain.Config{})

	req := httptest.NewRequest(http.MethodGet, "/keys/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
