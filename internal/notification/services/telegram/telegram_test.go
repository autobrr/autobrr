// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package telegram

import (
	"encoding/json"
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

	var got Message

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/botmock-token/sendMessage", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

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
