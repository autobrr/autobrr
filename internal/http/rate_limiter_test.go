// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
)

func TestClientRateLimiter(t *testing.T) {
	now := time.Now()

	t.Run("record blocks after burst and recovers", func(t *testing.T) {
		l := newClientRateLimiter(20*time.Second, 5)

		for range 5 {
			assert.Zero(t, l.retryAfter("192.0.2.1", now))
			l.record("192.0.2.1", now)
		}

		wait := l.retryAfter("192.0.2.1", now)
		assert.InDelta(t, 20*time.Second, wait, float64(time.Second))
		assert.Zero(t, l.retryAfter("192.0.2.2", now), "other clients are unaffected")
		assert.Zero(t, l.retryAfter("192.0.2.1", now.Add(wait)))
	})

	t.Run("reset clears the bucket", func(t *testing.T) {
		l := newClientRateLimiter(20*time.Second, 5)

		for range 5 {
			l.record("192.0.2.1", now)
		}
		l.reset("192.0.2.1")

		assert.Zero(t, l.retryAfter("192.0.2.1", now))
	})

	t.Run("take refuses without consuming", func(t *testing.T) {
		l := newClientRateLimiter(3*time.Second, 2)

		assert.Zero(t, l.take("192.0.2.1", now))
		assert.Zero(t, l.take("192.0.2.1", now))

		for range 10 {
			assert.InDelta(t, 3*time.Second, l.take("192.0.2.1", now), float64(time.Millisecond), "refused takes must not push the wait further out")
		}

		assert.Zero(t, l.take("192.0.2.1", now.Add(3*time.Second)))
	})
}

func TestClientRateLimiterMiddleware(t *testing.T) {
	key := func(r *http.Request) string { return clientIP(r.RemoteAddr) }
	handler := newClientRateLimiter(3*time.Second, 2).Middleware(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := func(remoteAddr string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = remoteAddr
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		return rec
	}

	assert.Equal(t, http.StatusNoContent, request("192.0.2.1:1000").Code)
	assert.Equal(t, http.StatusNoContent, request("192.0.2.1:1001").Code)

	rec := request("192.0.2.1:1002")
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "3", rec.Header().Get("Retry-After"))

	assert.Equal(t, http.StatusNoContent, request("192.0.2.2:1000").Code, "other clients are unaffected")
}

func TestServerLimiterKey(t *testing.T) {
	tests := []struct {
		name       string
		trusted    []netip.Prefix
		realIP     func(http.Handler) http.Handler
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{"peer without headers", nil, middleware.RealIP, "203.0.113.9:8888", nil, "203.0.113.9"},
		{"legacy RealIP rewrite is ignored", nil, middleware.RealIP, "203.0.113.9:8888", map[string]string{"X-Real-IP": "198.51.100.7"}, "203.0.113.9"},
		{"legacy RealIP XFF rewrite is ignored", nil, middleware.RealIP, "203.0.113.9:8888", map[string]string{"X-Forwarded-For": "198.51.100.7"}, "203.0.113.9"},
		{"IPv4-mapped peer is unmapped", nil, middleware.RealIP, "[::ffff:203.0.113.9]:8888", nil, "203.0.113.9"},
		{"untrusted peer keeps its address", []netip.Prefix{netip.MustParsePrefix("10.0.0.1/32")}, RealIPFromTrustedProxies([]netip.Prefix{netip.MustParsePrefix("10.0.0.1/32")}), "203.0.113.9:8888", map[string]string{"X-Forwarded-For": "198.51.100.7"}, "203.0.113.9"},
		{"trusted proxy resolves the client", []netip.Prefix{netip.MustParsePrefix("10.0.0.1/32")}, RealIPFromTrustedProxies([]netip.Prefix{netip.MustParsePrefix("10.0.0.1/32")}), "10.0.0.1:8888", map[string]string{"X-Forwarded-For": "198.51.100.7"}, "198.51.100.7"},
		{"IPv6 peer is grouped by /64", nil, middleware.RealIP, "[2001:db8:1:2:aaaa::1]:8888", nil, "2001:db8:1:2::/64"},
		{"IPv6 client behind a trusted proxy is grouped by /64", []netip.Prefix{netip.MustParsePrefix("10.0.0.1/32")}, RealIPFromTrustedProxies([]netip.Prefix{netip.MustParsePrefix("10.0.0.1/32")}), "10.0.0.1:8888", map[string]string{"X-Forwarded-For": "2001:db8:1:2::77"}, "2001:db8:1:2::/64"},
		{"trusted proxy without a chain keeps the peer", []netip.Prefix{netip.MustParsePrefix("10.0.0.1/32")}, RealIPFromTrustedProxies([]netip.Prefix{netip.MustParsePrefix("10.0.0.1/32")}), "10.0.0.1:8888", nil, "10.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{trustedProxyPrefixes: tt.trusted}

			var got string
			r := chi.NewRouter()
			r.Use(StorePeerAddr, tt.realIP)
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				got = s.limiterKey(r)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			r.ServeHTTP(httptest.NewRecorder(), req)

			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("falls back to the remote address without StorePeerAddr", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.1:51234"

		assert.Equal(t, "192.0.2.1", (&Server{}).limiterKey(req))
	})
}

func TestClientIP(t *testing.T) {
	assert.Equal(t, "192.0.2.1", clientIP("192.0.2.1:51234"))
	assert.Equal(t, "2001:db8::1", clientIP("[2001:db8::1]:51234"))
	assert.Equal(t, "192.0.2.1", clientIP("192.0.2.1"))
}
