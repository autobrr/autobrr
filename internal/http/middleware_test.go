// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func newAuthDisabledTestServer(cidrs []string) *Server {
	cfg := &domain.Config{
		AuthDisabled:               true,
		IAcknowledgeThisIsABadIdea: true,
		AuthDisabledAllowedCIDRs:   cidrs,
	}

	// mirrors what NewServer does: parse once up front rather than per request.
	prefixes, _ := cfg.ParseAuthDisabledAllowedCIDRs()

	return &Server{
		log:                         zerolog.Nop(),
		config:                      &config.AppConfig{Config: cfg},
		authDisabledAllowedPrefixes: prefixes,
	}
}

// newAuthDisabledTestServerWithLog is like newAuthDisabledTestServer but captures
// log output so tests can assert on *why* a request was rejected. The HTTP response
// itself intentionally stays a generic 403 - the reason is only ever logged
// server-side, never returned to the caller.
func newAuthDisabledTestServerWithLog(cidrs []string) (*Server, *bytes.Buffer) {
	var buf bytes.Buffer
	s := newAuthDisabledTestServer(cidrs)
	s.log = zerolog.New(&buf)
	return s, &buf
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireAuthDisabledIPAllowlist_PassThroughWhenAuthEnabled(t *testing.T) {
	s := &Server{
		log: zerolog.Nop(),
		config: &config.AppConfig{
			Config: &domain.Config{},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	rr := httptest.NewRecorder()

	s.RequireAuthDisabledIPAllowlist(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireAuthDisabledIPAllowlist_AllowsInRangeIP(t *testing.T) {
	s := newAuthDisabledTestServer([]string{"192.168.1.0/24"})

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.RemoteAddr = "192.168.1.42:5555"
	rr := httptest.NewRecorder()

	s.RequireAuthDisabledIPAllowlist(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireAuthDisabledIPAllowlist_BlocksOutOfRangeIP(t *testing.T) {
	s, logs := newAuthDisabledTestServerWithLog([]string{"192.168.1.0/24"})

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	rr := httptest.NewRecorder()

	s.RequireAuthDisabledIPAllowlist(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	// the reason is only ever logged server-side, never leaked to the response body.
	assert.Equal(t, "Forbidden\n", rr.Body.String())
	assert.Contains(t, logs.String(), "not in authDisabledAllowedCIDRs")
}

func TestRequireAuthDisabledIPAllowlist_LoopbackAllowedForHealthz(t *testing.T) {
	s := newAuthDisabledTestServer(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/healthz/liveness", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rr := httptest.NewRecorder()

	s.RequireAuthDisabledIPAllowlist(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireAuthDisabledIPAllowlist_IgnoresForwardedFor(t *testing.T) {
	// The allowlist check must run before middleware.RealIP, so it should key
	// off r.RemoteAddr, never X-Forwarded-For.
	s := newAuthDisabledTestServer([]string{"192.168.1.0/24"})

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("X-Forwarded-For", "192.168.1.42")
	rr := httptest.NewRecorder()

	s.RequireAuthDisabledIPAllowlist(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestIsAuthenticated_BypassedWhenAuthDisabled(t *testing.T) {
	s := newAuthDisabledTestServer([]string{"127.0.0.1/32"})

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	rr := httptest.NewRecorder()

	s.IsAuthenticated(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
