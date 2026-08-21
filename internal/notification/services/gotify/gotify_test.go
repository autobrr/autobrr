// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package gotify

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_SendMessage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/message", r.URL.Path)
		assert.Equal(t, "mock-token", r.Header.Get("X-Gotify-Key"))
		assert.Empty(t, r.URL.Query().Get("token"))
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		require.NoError(t, r.ParseForm())
		assert.Equal(t, "Push Approved", r.PostForm.Get("title"))
		assert.Equal(t, "New release: Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP\n", r.PostForm.Get("message"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Host: server.URL, Token: "mock-token", Name: "mock"})

	message := &Message{
		Title:   "Push Approved",
		Message: "New release: Best.Show.Ever.S18E21.1080p.AMZN.WEB-DL.DDP2.0.H.264-GROUP\n",
	}

	assert.NoError(t, client.SendMessage(t.Context(), message))
}

func TestClient_SendMessage_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Unauthorized","errorCode":401,"errorDescription":"you need to provide a valid access token or user credentials to access this api"}`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Host: server.URL, Token: "mock-token", Name: "mock"})

	err := client.SendMessage(t.Context(), &Message{Title: "Test", Message: "autobrr goes brr!!"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "check gotify application token")
		assert.Contains(t, err.Error(), "unexpected status: 401")
		assert.Contains(t, err.Error(), "Unauthorized")
		assert.Contains(t, err.Error(), "you need to provide a valid access token")
	}
}

func TestClient_SendMessage_ErrorRawBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`bad gateway`))
	}))
	defer server.Close()

	client := NewSender(zerolog.New(io.Discard), Config{Host: server.URL, Token: "mock-token", Name: "mock"})

	err := client.SendMessage(t.Context(), &Message{Title: "Test", Message: "autobrr goes brr!!"})
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "unexpected status: 502")
		assert.Contains(t, err.Error(), "bad gateway")
	}
}
