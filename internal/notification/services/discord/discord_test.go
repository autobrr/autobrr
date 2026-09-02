// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package discord

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

func testMessage() *Message {
	message := NewMessage()
	message.Content = "New release: Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP"

	return message
}

func TestClient_SendMessage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "autobrr", r.Header.Get("User-Agent"))

		var payload Message
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "New release: Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP", payload.Content)

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{WebHookURL: server.URL, Name: "mock"})

	assert.NoError(t, client.SendMessage(t.Context(), testMessage()))
}

func TestClient_SendMessage_RateLimitedThenSuccess(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.NotEmpty(t, body, "retried request must not have an empty body")

		if requests.Add(1) == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message": "You are being rate limited.", "retry_after": 0.3, "global": false}`))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{WebHookURL: server.URL, Name: "mock"})

	start := time.Now()
	assert.NoError(t, client.SendMessage(t.Context(), testMessage()))
	assert.GreaterOrEqual(t, time.Since(start), time.Millisecond*270, "retry must honor the server-directed delay")
	assert.Equal(t, int32(2), requests.Load())
}

func TestClient_SendMessage_RateLimitTooLong(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message": "You are being rate limited.", "retry_after": 120.5, "global": true}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{WebHookURL: server.URL, Name: "mock"})

	err := client.SendMessage(t.Context(), testMessage())
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unexpected status: 429")
		assert.Contains(t, err.Error(), "You are being rate limited.")
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
			_, _ = w.Write([]byte(`{"message": "You are being rate limited.", "global": false}`))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{WebHookURL: server.URL, Name: "mock"})

	start := time.Now()
	assert.NoError(t, client.SendMessage(t.Context(), testMessage()))
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
		_, _ = w.Write([]byte(`{"message": "You are being rate limited.", "global": true}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{WebHookURL: server.URL, Name: "mock"})

	err := client.SendMessage(t.Context(), testMessage())
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unexpected status: 429")
		assert.Contains(t, err.Error(), "exceeds the retry budget")
	}
	assert.Equal(t, int32(1), requests.Load())
}

func TestClient_SendMessage_BadRequestNoRetry(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message": "Cannot send an empty message", "code": 50006}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{WebHookURL: server.URL, Name: "mock"})

	err := client.SendMessage(t.Context(), testMessage())
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "status: 400")
		assert.Contains(t, err.Error(), "Cannot send an empty message")
		assert.Contains(t, err.Error(), "50006")
	}
	assert.Equal(t, int32(1), requests.Load())
}

func TestClient_SendMessage_NotFoundNoRetry(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message": "Unknown Webhook", "code": 10015}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{WebHookURL: server.URL, Name: "mock"})

	err := client.SendMessage(t.Context(), testMessage())
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "Unknown Webhook")
		assert.Contains(t, err.Error(), "check the webhook URL")
	}
	assert.Equal(t, int32(1), requests.Load())
}

func TestClient_SendMessage_ServerErrorRetries(t *testing.T) {
	shortenRetryDelay(t)

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`upstream exploded`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{WebHookURL: server.URL, Name: "mock"})

	err := client.SendMessage(t.Context(), testMessage())
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "status: 500")
		assert.Contains(t, err.Error(), "upstream exploded")
	}
	assert.Equal(t, int32(3), requests.Load())
}
