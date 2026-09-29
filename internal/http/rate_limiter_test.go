// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	handler := newClientRateLimiter(3*time.Second, 2).Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestClientIP(t *testing.T) {
	assert.Equal(t, "192.0.2.1", clientIP("192.0.2.1:51234"))
	assert.Equal(t, "2001:db8::1", clientIP("[2001:db8::1]:51234"))
	assert.Equal(t, "192.0.2.1", clientIP("192.0.2.1"))
}
