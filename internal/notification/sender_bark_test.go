// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestBarkSender_Send(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]any{
			"device_key": "mock-key",
			"title":      "Autobrr update available",
			"body":       "New update available!\nv1.2.0\n",
			"sound":      "minuet",
			"group":      "autobrr",
			"level":      "timeSensitive",
			"icon":       defaultImageURL,
			"url":        "https://github.com/autobrr/autobrr/releases/tag/v1.2.0",
		}, body)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewBarkSender(zerolog.New(io.Discard), &domain.Notification{
		Name:     "Bark",
		Type:     domain.NotificationTypeBark,
		Enabled:  true,
		Host:     server.URL,
		Token:    "mock-key",
		Sound:    "minuet",
		Topic:    "autobrr",
		Priority: 1,
	})

	assert.True(t, sender.IsEnabled())
	assert.NoError(t, sender.Send(t.Context(), domain.NotificationPayload{
		Event:   domain.NotificationEventAppUpdateAvailable,
		Subject: "New update available!",
		Message: "v1.2.0",
		URL:     "https://github.com/autobrr/autobrr/releases/tag/v1.2.0",
	}))
}

func TestBarkSender_Send_Level(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		priority  int32
		wantLevel any
	}{
		{name: "passive", priority: -1, wantLevel: "passive"},
		{name: "default", priority: 0, wantLevel: nil},
		{name: "time sensitive", priority: 1, wantLevel: "timeSensitive"},
		{name: "critical", priority: 2, wantLevel: "critical"},
		// a priority left over from another service type falls back to the Bark default
		{name: "unknown", priority: 3, wantLevel: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, tt.wantLevel, body["level"])

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			sender := NewBarkSender(zerolog.New(io.Discard), &domain.Notification{
				Type:     domain.NotificationTypeBark,
				Host:     server.URL,
				Token:    "mock-key",
				Priority: tt.priority,
			})

			assert.NoError(t, sender.Send(t.Context(), domain.NotificationPayload{Event: domain.NotificationEventTest}))
		})
	}
}
