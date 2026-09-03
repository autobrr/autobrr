// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package pushover

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_SendMessage(t *testing.T) {
	t.Parallel()

	sent := time.Now()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		require.NoError(t, r.ParseForm())
		assert.Equal(t, "mock-token", r.PostForm.Get("token"))
		assert.Equal(t, "mock-user", r.PostForm.Get("user"))
		assert.Equal(t, "Push Approved", r.PostForm.Get("title"))
		assert.Equal(t, "New release: Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP\n", r.PostForm.Get("message"))
		assert.Equal(t, strconv.FormatInt(sent.Unix(), 10), r.PostForm.Get("timestamp"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":1,"request":"mock-request-id"}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Token: "mock-token", User: "mock-user", Name: "mock"})
	client.endpoint = server.URL

	message := &Message{
		Title:     "Push Approved",
		Message:   "New release: Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP\n",
		Timestamp: sent,
	}

	assert.NoError(t, client.SendMessage(t.Context(), message))
}

func TestClient_SendMessage_ZeroTimestampNotSent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.False(t, r.PostForm.Has("timestamp"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Token: "mock-token", User: "mock-user", Name: "mock"})
	client.endpoint = server.URL

	assert.NoError(t, client.SendMessage(t.Context(), &Message{Title: "Test", Message: "autobrr goes brr!!"}))
}

func TestClient_SendMessage_QuotaExhaustedNoRetry(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"status":0,"request":"mock-request-id","errors":["message quota exceeded"]}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Token: "mock-token", User: "mock-user", Name: "mock"})
	client.endpoint = server.URL

	err := client.SendMessage(t.Context(), &Message{Title: "Test", Message: "autobrr goes brr!!"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "monthly message quota exhausted")
		assert.Contains(t, err.Error(), "unexpected status: 429")
	}
	assert.Equal(t, int32(1), requests.Load())
}

func TestClient_SendMessage_BadRequestNoRetry(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":0,"request":"mock-request-id","user":"invalid","errors":["user identifier is not a valid user, group, or subscribed user key"]}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Token: "mock-token", User: "mock-user", Name: "mock"})
	client.endpoint = server.URL

	err := client.SendMessage(t.Context(), &Message{Title: "Test", Message: "autobrr goes brr!!"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unexpected status: 400")
		assert.Contains(t, err.Error(), "user identifier is not a valid user, group, or subscribed user key")
	}
	assert.Equal(t, int32(1), requests.Load())
}

// shortenRetryDelay must only be used in sequential tests: parallel tests are
// paused until every sequential test, including the cleanup, has finished.
func shortenRetryDelay(t *testing.T) {
	t.Helper()

	prev := retryDelay
	retryDelay = time.Millisecond * 10
	t.Cleanup(func() { retryDelay = prev })
}

func TestClient_SendMessage_ServerErrorRetries(t *testing.T) {
	shortenRetryDelay(t)

	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, "autobrr goes brr!!", r.PostForm.Get("message"))

		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Token: "mock-token", User: "mock-user", Name: "mock"})
	client.endpoint = server.URL

	err := client.SendMessage(t.Context(), &Message{Title: "Test", Message: "autobrr goes brr!!"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unexpected status: 500")
	}
	assert.Equal(t, int32(3), requests.Load())
}
