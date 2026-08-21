// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package ntfy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shortenRetryDelay must only be used in sequential tests: parallel tests are
// paused until every sequential test, including the cleanup, has finished.
func shortenRetryDelay(t *testing.T) {
	t.Helper()

	prevRetryDelay, prevDefaultRetryAfter := retryDelay, defaultRetryAfter
	retryDelay = time.Millisecond * 10
	defaultRetryAfter = time.Millisecond * 10
	t.Cleanup(func() { retryDelay, defaultRetryAfter = prevRetryDelay, prevDefaultRetryAfter })
}

func TestClient_SendMessage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/autobrr", r.URL.Path)
		assert.Equal(t, "text/plain", r.Header.Get("Content-Type"))
		assert.Equal(t, "Push Approved", r.Header.Get("Title"))
		assert.Equal(t, "3", r.Header.Get("Priority"))
		assert.Equal(t, "tada", r.Header.Get("Tags"))
		assert.Equal(t, "Bearer mock-token", r.Header.Get("Authorization"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "New release: Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP\n", string(body))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Host: server.URL + "/autobrr", Token: "mock-token", Name: "mock"})

	message := &Message{
		Title:    "Push Approved",
		Message:  "New release: Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP\n",
		Priority: 3,
		Tags:     "tada",
	}

	assert.NoError(t, client.SendMessage(t.Context(), message))
}

// Basic auth takes precedence over the access token.
func TestClient_SendMessage_BasicAuth(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "mock-user", username)
		assert.Equal(t, "mock-pass", password)
		assert.Empty(t, r.Header.Get("Priority"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{Host: server.URL, Token: "mock-token", Username: "mock-user", Password: "mock-pass", Name: "mock"}
	client := NewSender(zerolog.New(io.Discard), config)

	assert.NoError(t, client.SendMessage(t.Context(), &Message{Title: "Test", Message: "autobrr goes brr!!"}))
}

func TestClient_SendMessage_RateLimitedThenSuccess(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "autobrr goes brr!!", string(body))

		if requests.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"code":42901,"http":429,"error":"limit reached"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Host: server.URL, Name: "mock"})

	start := time.Now()
	assert.NoError(t, client.SendMessage(t.Context(), &Message{Title: "Test", Message: "autobrr goes brr!!"}))
	assert.GreaterOrEqual(t, time.Since(start), time.Millisecond*900, "retry must honor the server-directed delay")
	assert.Equal(t, int32(2), requests.Load())
}

func TestClient_SendMessage_RateLimitFallbackDelay(t *testing.T) {
	shortenRetryDelay(t)

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"code":42901,"http":429,"error":"limit reached"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Host: server.URL, Name: "mock"})

	assert.NoError(t, client.SendMessage(t.Context(), &Message{Title: "Test", Message: "autobrr goes brr!!"}))
	assert.Equal(t, int32(2), requests.Load())
}

func TestClient_SendMessage_RateLimitTooLong(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"code":42901,"http":429,"error":"limit reached"}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Host: server.URL, Name: "mock"})

	err := client.SendMessage(t.Context(), &Message{Title: "Test", Message: "autobrr goes brr!!"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unexpected status: 429")
		assert.Contains(t, err.Error(), "exceeds the retry budget")
	}
	assert.Equal(t, int32(1), requests.Load())
}

func TestClient_SendMessage_RateLimitHTTPDate(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Retry-After", time.Now().Add(time.Second*2).UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"code":42901,"http":429,"error":"limit reached"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Host: server.URL, Name: "mock"})

	start := time.Now()
	assert.NoError(t, client.SendMessage(t.Context(), &Message{Title: "Test", Message: "autobrr goes brr!!"}))
	assert.GreaterOrEqual(t, time.Since(start), time.Millisecond*900, "retry must honor an HTTP-date Retry-After")
	assert.Equal(t, int32(2), requests.Load())
}

func TestClient_SendMessage_RateLimitHTTPDateTooLong(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Retry-After", time.Now().Add(time.Minute).UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"code":42901,"http":429,"error":"limit reached"}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Host: server.URL, Name: "mock"})

	err := client.SendMessage(t.Context(), &Message{Title: "Test", Message: "autobrr goes brr!!"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unexpected status: 429")
		assert.Contains(t, err.Error(), "exceeds the retry budget")
	}
	assert.Equal(t, int32(1), requests.Load())
}

func TestClient_SendMessage_ServerErrorRetries(t *testing.T) {
	shortenRetryDelay(t)

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Host: server.URL, Name: "mock"})

	err := client.SendMessage(t.Context(), &Message{Title: "Test", Message: "autobrr goes brr!!"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unexpected status: 500")
	}
	assert.Equal(t, int32(3), requests.Load())
}

func TestClient_SendMessage_UnauthorizedNoRetry(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":40101,"http":401,"error":"unauthorized"}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Host: server.URL, Token: "bad-token", Name: "mock"})

	err := client.SendMessage(t.Context(), &Message{Title: "Test", Message: "autobrr goes brr!!"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "check access token or username/password")
		assert.Contains(t, err.Error(), "unexpected status: 401")
		assert.Contains(t, err.Error(), "unauthorized")
	}
	assert.Equal(t, int32(1), requests.Load())
}
