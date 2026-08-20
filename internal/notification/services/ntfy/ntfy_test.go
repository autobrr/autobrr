// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package ntfy

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
