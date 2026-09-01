// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func newAuthDisabledTestServer(cidrs []string) *Server {
	cfg := &domain.Config{
		AuthDisabled:                true,
		AuthDisabledAcknowledgement: domain.AuthDisabledAcknowledgementValue,
		AuthAllowedPeerCIDRs:        cidrs,
	}

	// mirrors what NewServer does: parse once up front rather than per request.
	prefixes, _ := cfg.ParseAuthAllowedPeerCIDRs()

	return &Server{
		log:                     zerolog.Nop(),
		config:                  &config.AppConfig{Config: cfg},
		authAllowedPeerPrefixes: prefixes,
		healthCheckPaths:        []string{"/api/healthz/liveness", "/api/healthz/readiness"},
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
	assert.Contains(t, logs.String(), "not in authAllowedPeerCIDRs")
}

func TestRequireAuthDisabledIPAllowlist_AllowsIPv6Peer(t *testing.T) {
	s := newAuthDisabledTestServer([]string{"2001:db8::/32"})

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.RemoteAddr = "[2001:db8::5]:1234"
	rr := httptest.NewRecorder()

	s.RequireAuthDisabledIPAllowlist(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireAuthDisabledIPAllowlist_BlocksIPv6PeerAgainstIPv4List(t *testing.T) {
	s := newAuthDisabledTestServer([]string{"192.168.1.0/24"})

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.RemoteAddr = "[2001:db8::5]:1234"
	rr := httptest.NewRecorder()

	s.RequireAuthDisabledIPAllowlist(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireAuthDisabledIPAllowlist_AllowsIPv4MappedIPv6Peer(t *testing.T) {
	// A dual-stack peer may arrive as ::ffff:192.168.1.42; it must be normalized
	// so it still matches an IPv4 allowlist entry.
	s := newAuthDisabledTestServer([]string{"192.168.1.0/24"})

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.RemoteAddr = "[::ffff:192.168.1.42]:1234"
	rr := httptest.NewRecorder()

	s.RequireAuthDisabledIPAllowlist(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireAuthDisabledIPAllowlist_BlocksMalformedRemoteAddr(t *testing.T) {
	s := newAuthDisabledTestServer([]string{"192.168.1.0/24"})

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.RemoteAddr = "not-an-ip-address"
	rr := httptest.NewRecorder()

	s.RequireAuthDisabledIPAllowlist(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireAuthDisabledIPAllowlist_DeniesWhenPrefixListEmpty(t *testing.T) {
	s := newAuthDisabledTestServer(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rr := httptest.NewRecorder()

	s.RequireAuthDisabledIPAllowlist(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireAuthDisabledIPAllowlist_LoopbackAllowedForHealthz(t *testing.T) {
	s := newAuthDisabledTestServer(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/healthz/liveness", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rr := httptest.NewRecorder()

	s.RequireAuthDisabledIPAllowlist(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireAuthDisabledIPAllowlist_HealthzExceptionIsExactPathOnly(t *testing.T) {
	// A path that merely ends in the health path must not inherit the loopback
	// exception; only the exact resolved path qualifies.
	s := newAuthDisabledTestServer(nil)

	req := httptest.NewRequest(http.MethodGet, "/evil/api/healthz/liveness", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rr := httptest.NewRecorder()

	s.RequireAuthDisabledIPAllowlist(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireAuthDisabledIPAllowlist_HealthzExceptionOnlyForSafeMethods(t *testing.T) {
	s := newAuthDisabledTestServer(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/healthz/liveness", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rr := httptest.NewRecorder()

	s.RequireAuthDisabledIPAllowlist(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
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

func TestRequireAuthDisabledIPAllowlist_IgnoresSpoofedSingleIPHeaders(t *testing.T) {
	// None of the single-IP forwarding headers may move a denied peer into the
	// allowlist, regardless of which one a proxy/attacker sets.
	for _, header := range []string{"X-Real-IP", "True-Client-IP", "CF-Connecting-IP"} {
		t.Run(header, func(t *testing.T) {
			s := newAuthDisabledTestServer([]string{"192.168.1.0/24"})

			req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
			req.RemoteAddr = "203.0.113.5:1234"
			req.Header.Set(header, "192.168.1.42")
			rr := httptest.NewRecorder()

			s.RequireAuthDisabledIPAllowlist(okHandler()).ServeHTTP(rr, req)

			assert.Equal(t, http.StatusForbidden, rr.Code)
		})
	}
}

func TestRequireAuthDisabledIPAllowlist_RunsBeforeRealIP(t *testing.T) {
	// Pin the load-bearing ordering: even with middleware.RealIP in the chain, the
	// allowlist runs first on the true peer, so a spoofed X-Real-IP can't get in.
	s := newAuthDisabledTestServer([]string{"192.168.1.0/24"})

	handler := s.RequireAuthDisabledIPAllowlist(middleware.RealIP(okHandler()))

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("X-Real-IP", "192.168.1.42")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireAuthDisabledIPAllowlist_ClientIPHeaderCannotBypass(t *testing.T) {
	// The configured client IP header is for logging only; the allowlist keys off
	// the TCP peer even with ClientIPFromHeader in the chain behind it.
	s := newAuthDisabledTestServer([]string{"192.168.1.0/24"})

	handler := s.RequireAuthDisabledIPAllowlist(middleware.ClientIPFromHeader("X-Real-IP")(okHandler()))

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("X-Real-IP", "192.168.1.42")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestLoggerMiddleware_LogsResolvedClientIP(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	handler := middleware.ClientIPFromHeader("X-Real-IP")(LoggerMiddleware(&logger, nil)(okHandler()))

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Real-IP", "192.0.2.7")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Contains(t, buf.String(), `"remote_ip":"192.0.2.7"`)
	// ClientIPFromHeader must never rewrite the peer address itself.
	assert.Equal(t, "127.0.0.1:1234", req.RemoteAddr)
}

func TestLoggerMiddleware_FallsBackToRemoteAddr(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)

	handler := LoggerMiddleware(&logger, nil)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Contains(t, buf.String(), `"remote_ip":"127.0.0.1:1234"`)
}

// newAuthDisabledTestServerWithOrigins mirrors NewServer's origin-set derivation
// on top of newAuthDisabledTestServer.
func newAuthDisabledTestServerWithOrigins(cidrs []string, corsOrigins string) *Server {
	s := newAuthDisabledTestServer(cidrs)
	s.config.Config.CorsAllowedOrigins = corsOrigins

	if err := s.buildAuthAllowedOriginSets(); err != nil {
		panic(err)
	}

	return s
}

func TestRejectUntrustedBrowserRequests_PassThroughWhenAuthEnabled(t *testing.T) {
	s := &Server{
		log:    zerolog.Nop(),
		config: &config.AppConfig{Config: &domain.Config{}},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/filters", strings.NewReader(`{}`))
	req.Header.Set("Origin", "https://evil.example.com")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rr := httptest.NewRecorder()

	s.RejectUntrustedBrowserRequests(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRejectUntrustedBrowserRequests_AllowsRequestsWithoutBrowserMarkers(t *testing.T) {
	// curl-style API clients send no Origin or Sec-Fetch headers and any Host.
	s := newAuthDisabledTestServerWithOrigins([]string{"127.0.0.1/32"}, "https://autobrr.example.com")

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.Host = "nas:7474"
	rr := httptest.NewRecorder()

	s.RejectUntrustedBrowserRequests(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRejectUntrustedBrowserRequests_AllowsAllowlistedOrigin(t *testing.T) {
	s := newAuthDisabledTestServerWithOrigins([]string{"127.0.0.1/32"}, "https://autobrr.example.com")

	req := httptest.NewRequest(http.MethodPost, "/api/filters", strings.NewReader(`{}`))
	req.Host = "autobrr.example.com"
	req.Header.Set("Origin", "https://autobrr.example.com")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.RejectUntrustedBrowserRequests(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRejectUntrustedBrowserRequests_BlocksDisallowedOrigin(t *testing.T) {
	s := newAuthDisabledTestServerWithOrigins([]string{"127.0.0.1/32"}, "https://autobrr.example.com")

	req := httptest.NewRequest(http.MethodPost, "/api/filters", strings.NewReader(`{}`))
	req.Host = "autobrr.example.com"
	req.Header.Set("Origin", "https://evil.example.com")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.RejectUntrustedBrowserRequests(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRejectUntrustedBrowserRequests_BlocksCrossSiteNavigation(t *testing.T) {
	// A GET navigation (link click) carries Sec-Fetch-Site but no Origin; it must
	// not reach state-changing GET routes like the webhook triggers.
	s := newAuthDisabledTestServerWithOrigins([]string{"127.0.0.1/32"}, "https://autobrr.example.com")

	req := httptest.NewRequest(http.MethodGet, "/api/webhook/lists/trigger", nil)
	req.Host = "autobrr.example.com"
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	rr := httptest.NewRecorder()

	s.RejectUntrustedBrowserRequests(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRejectUntrustedBrowserRequests_AllowsCrossSiteFetchFromAllowlistedOrigin(t *testing.T) {
	// A dashboard on another domain doing authorized fetch calls is exactly what
	// corsAllowedOrigins exists for; an allowlisted Origin overrides Sec-Fetch.
	s := newAuthDisabledTestServerWithOrigins([]string{"127.0.0.1/32"}, "https://autobrr.example.com,https://dash.example.net")

	req := httptest.NewRequest(http.MethodPost, "/api/filters", strings.NewReader(`{}`))
	req.Host = "autobrr.example.com"
	req.Header.Set("Origin", "https://dash.example.net")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	s.RejectUntrustedBrowserRequests(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRejectUntrustedBrowserRequests_AllowsNonCrossSiteFetchValues(t *testing.T) {
	// "none" is a user-initiated navigation (address bar, bookmark) and
	// "same-site" covers sibling subdomains; neither is an attack vector here.
	for _, site := range []string{"same-origin", "same-site", "none"} {
		t.Run(site, func(t *testing.T) {
			s := newAuthDisabledTestServerWithOrigins([]string{"127.0.0.1/32"}, "https://autobrr.example.com")

			req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
			req.Host = "autobrr.example.com"
			req.Header.Set("Sec-Fetch-Site", site)
			rr := httptest.NewRecorder()

			s.RejectUntrustedBrowserRequests(okHandler()).ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestRejectUntrustedBrowserRequests_BlocksUnknownHostForBrowserRequests(t *testing.T) {
	// DNS rebinding: the rebound request looks same-origin to the browser, but the
	// Host header still names the attacker's domain.
	s := newAuthDisabledTestServerWithOrigins([]string{"127.0.0.1/32"}, "https://autobrr.example.com")

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	req.Host = "rebind.evil.example"
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rr := httptest.NewRecorder()

	s.RejectUntrustedBrowserRequests(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRejectUntrustedBrowserRequests_AllowsIPLiteralAndLocalhostHosts(t *testing.T) {
	// Rebinding requires an attacker-controlled DNS name, so direct-IP and
	// localhost access needs no extra configuration.
	for _, host := range []string{"192.168.1.10:7474", "[::1]:7474", "localhost:7474", "localhost"} {
		t.Run(host, func(t *testing.T) {
			s := newAuthDisabledTestServerWithOrigins([]string{"127.0.0.1/32"}, "https://autobrr.example.com")

			req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
			req.Host = host
			req.Header.Set("Sec-Fetch-Site", "same-origin")
			rr := httptest.NewRecorder()

			s.RejectUntrustedBrowserRequests(okHandler()).ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)
		})
	}
}

func TestRejectUntrustedBrowserRequests_BlocksTextPlainMutation(t *testing.T) {
	// A cross-site form can submit JSON in a text/plain body without a preflight.
	s := newAuthDisabledTestServerWithOrigins([]string{"127.0.0.1/32"}, "https://autobrr.example.com")

	req := httptest.NewRequest(http.MethodPost, "/api/filters", strings.NewReader(`{"name":"x"}`))
	req.Host = "autobrr.example.com"
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	s.RejectUntrustedBrowserRequests(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
}

func TestRejectUntrustedBrowserRequests_AllowsJSONContentTypeWithCharset(t *testing.T) {
	s := newAuthDisabledTestServerWithOrigins([]string{"127.0.0.1/32"}, "https://autobrr.example.com")

	req := httptest.NewRequest(http.MethodPost, "/api/filters", strings.NewReader(`{"name":"x"}`))
	req.Host = "autobrr.example.com"
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rr := httptest.NewRecorder()

	s.RejectUntrustedBrowserRequests(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRejectUntrustedBrowserRequests_AllowsBodylessMutationWithoutContentType(t *testing.T) {
	// The webhook trigger POSTs take no body; they must keep working from plain
	// API clients that set no Content-Type at all.
	s := newAuthDisabledTestServerWithOrigins([]string{"127.0.0.1/32"}, "https://autobrr.example.com")

	req := httptest.NewRequest(http.MethodPost, "/api/webhook/lists/trigger", nil)
	req.Host = "nas:7474"
	rr := httptest.NewRecorder()

	s.RejectUntrustedBrowserRequests(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestIsAuthenticated_BypassedWhenAuthDisabled(t *testing.T) {
	s := newAuthDisabledTestServer([]string{"127.0.0.1/32"})

	req := httptest.NewRequest(http.MethodGet, "/api/filters", nil)
	rr := httptest.NewRecorder()

	s.IsAuthenticated(okHandler()).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
