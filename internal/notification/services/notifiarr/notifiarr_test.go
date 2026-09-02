// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notifiarr

import (
	"encoding/json"
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

	var got Message

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "mock-api-key", r.Header.Get("X-API-Key"))
		assert.Equal(t, "autobrr", r.Header.Get("User-Agent"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &got))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"success"}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{APIKey: "mock-api-key", Name: "mock"})
	client.endpoint = server.URL

	message := &Message{
		Event: "PUSH_APPROVED",
		Data: MessageData{
			Subject: "Push Approved",
			Message: "New release: Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP\n",
			Event:   "PUSH_APPROVED",
		},
	}

	assert.NoError(t, client.SendMessage(t.Context(), message))
	assert.Equal(t, *message, got)
}

func TestClient_SendMessage_RateLimitedThenSuccess(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	var got Message

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"result":"error","details":{"response":"rate limited"}}`))
			return
		}

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &got))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"success"}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{APIKey: "mock-api-key", Name: "mock"})
	client.endpoint = server.URL

	message := &Message{Event: "TEST", Data: MessageData{Subject: "Test", Message: "autobrr goes brr!!"}}

	start := time.Now()
	assert.NoError(t, client.SendMessage(t.Context(), message))
	assert.GreaterOrEqual(t, time.Since(start), time.Millisecond*900, "retry must honor the server-directed delay")
	assert.Equal(t, int32(2), requests.Load())
	assert.Equal(t, "autobrr goes brr!!", got.Data.Message)
}

func TestClient_SendMessage_RateLimitFallbackDelay(t *testing.T) {
	shortenRetryDelay(t)

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"result":"error","details":{"response":"rate limited"}}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"success"}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{APIKey: "mock-api-key", Name: "mock"})
	client.endpoint = server.URL

	assert.NoError(t, client.SendMessage(t.Context(), &Message{Event: "TEST"}))
	assert.Equal(t, int32(2), requests.Load())
}

func TestClient_SendMessage_RateLimitTooLong(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{APIKey: "mock-api-key", Name: "mock"})
	client.endpoint = server.URL

	err := client.SendMessage(t.Context(), &Message{Event: "TEST"})
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
			_, _ = w.Write([]byte(`{"result":"error","details":{"response":"rate limited"}}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"success"}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{APIKey: "mock-api-key", Name: "mock"})
	client.endpoint = server.URL

	start := time.Now()
	assert.NoError(t, client.SendMessage(t.Context(), &Message{Event: "TEST"}))
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
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{APIKey: "mock-api-key", Name: "mock"})
	client.endpoint = server.URL

	err := client.SendMessage(t.Context(), &Message{Event: "TEST"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unexpected status: 429")
		assert.Contains(t, err.Error(), "exceeds the retry budget")
	}
	assert.Equal(t, int32(1), requests.Load())
}

func TestClient_SendMessage_UnauthorizedNoRetry(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"result":"error","details":{"response":"invalid api key"}}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{APIKey: "mock-api-key", Name: "mock"})
	client.endpoint = server.URL

	err := client.SendMessage(t.Context(), &Message{Event: "TEST"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unexpected status: 401")
		assert.Contains(t, err.Error(), "invalid api key")
		assert.Contains(t, err.Error(), "check notifiarr api key")
	}
	assert.Equal(t, int32(1), requests.Load())
}

func TestClient_SendMessage_RequestTimeoutRetries(t *testing.T) {
	shortenRetryDelay(t)

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusRequestTimeout)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"success"}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{APIKey: "mock-api-key", Name: "mock"})
	client.endpoint = server.URL

	assert.NoError(t, client.SendMessage(t.Context(), &Message{Event: "TEST"}))
	assert.Equal(t, int32(2), requests.Load())
}

func TestClient_SendMessage_ServerErrorRetries(t *testing.T) {
	shortenRetryDelay(t)

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{APIKey: "mock-api-key", Name: "mock"})
	client.endpoint = server.URL

	err := client.SendMessage(t.Context(), &Message{Event: "TEST"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unexpected status: 502")
	}
	assert.Equal(t, int32(3), requests.Load())
}
