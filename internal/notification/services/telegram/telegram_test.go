// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package telegram

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
		assert.Equal(t, "/botmock-token/sendMessage", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "autobrr", r.Header.Get("User-Agent"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &got))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{Host: server.URL, Token: "mock-token", ChatID: "-100123456789", ThreadID: 21, Name: "mock"}
	client := NewSender(zerolog.New(io.Discard), config)

	message := &Message{
		Text:      "<b>New release:</b> Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP\n",
		ParseMode: "HTML",
	}

	assert.NoError(t, client.SendMessage(t.Context(), message))

	want := Message{
		ChatID:          "-100123456789",
		Text:            message.Text,
		ParseMode:       "HTML",
		MessageThreadID: 21,
	}
	assert.Equal(t, want, got)
}

func TestClient_SendMessage_RateLimitedThenSuccess(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	var got Message

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 1","parameters":{"retry_after":1}}`))
			return
		}

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &got))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{Host: server.URL, Token: "mock-token", ChatID: "-100123456789", Name: "mock"}
	client := NewSender(zerolog.New(io.Discard), config)

	start := time.Now()
	assert.NoError(t, client.SendMessage(t.Context(), &Message{Text: "autobrr goes brr!!"}))
	assert.GreaterOrEqual(t, time.Since(start), time.Millisecond*900, "retry must honor the server-directed delay")
	assert.Equal(t, int32(2), requests.Load())
	assert.Equal(t, "autobrr goes brr!!", got.Text)
}

func TestClient_SendMessage_RateLimitFallbackDelay(t *testing.T) {
	shortenRetryDelay(t)

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 0"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{Host: server.URL, Token: "mock-token", ChatID: "-100123456789", Name: "mock"}
	client := NewSender(zerolog.New(io.Discard), config)

	assert.NoError(t, client.SendMessage(t.Context(), &Message{Text: "autobrr goes brr!!"}))
	assert.Equal(t, int32(2), requests.Load())
}

func TestClient_SendMessage_RateLimitTooLong(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 30","parameters":{"retry_after":30}}`))
	}))
	defer server.Close()

	config := Config{Host: server.URL, Token: "mock-token", ChatID: "-100123456789", Name: "mock"}
	client := NewSender(zerolog.New(io.Discard), config)

	err := client.SendMessage(t.Context(), &Message{Text: "autobrr goes brr!!"})
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
			_, _ = w.Write([]byte(`{"ok":false,"error_code":429,"description":"Too Many Requests"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := Config{Host: server.URL, Token: "mock-token", ChatID: "-100123456789", Name: "mock"}
	client := NewSender(zerolog.New(io.Discard), config)

	start := time.Now()
	assert.NoError(t, client.SendMessage(t.Context(), &Message{Text: "autobrr goes brr!!"}))
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
		_, _ = w.Write([]byte(`{"ok":false,"error_code":429,"description":"Too Many Requests"}`))
	}))
	defer server.Close()

	config := Config{Host: server.URL, Token: "mock-token", ChatID: "-100123456789", Name: "mock"}
	client := NewSender(zerolog.New(io.Discard), config)

	err := client.SendMessage(t.Context(), &Message{Text: "autobrr goes brr!!"})
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
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`))
	}))
	defer server.Close()

	config := Config{Host: server.URL, Token: "mock-token", ChatID: "-100123456789", Name: "mock"}
	client := NewSender(zerolog.New(io.Discard), config)

	err := client.SendMessage(t.Context(), &Message{Text: "autobrr goes brr!!"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unexpected status: 400")
		assert.Contains(t, err.Error(), "Bad Request: chat not found")
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

	config := Config{Host: server.URL, Token: "mock-token", ChatID: "-100123456789", Name: "mock"}
	client := NewSender(zerolog.New(io.Discard), config)

	err := client.SendMessage(t.Context(), &Message{Text: "autobrr goes brr!!"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unexpected status: 500")
	}
	assert.Equal(t, int32(3), requests.Load())
}

func TestClient_SendMessage_MalformedHostDoesNotLeakToken(t *testing.T) {
	t.Parallel()

	config := Config{Host: "https://example.com/%zz", Token: "secret-token", ChatID: "-100123456789", Name: "mock"}
	client := NewSender(zerolog.New(io.Discard), config)

	err := client.SendMessage(t.Context(), &Message{Text: "autobrr goes brr!!"})
	if assert.Error(t, err) {
		assert.NotContains(t, err.Error(), "secret-token")
	}
}
