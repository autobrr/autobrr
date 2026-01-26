// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenericWebhookSender_Send(t *testing.T) {
	// Create a mock server to receive the webhook
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "autobrr", r.Header.Get("User-Agent"))
		assert.Equal(t, string(domain.WebhookEventReleaseNew), r.Header.Get("X-Autobrr-Event"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload domain.WebhookEvent
		err = json.Unmarshal(body, &payload)
		require.NoError(t, err)

		// Assert structured payload
		assert.Equal(t, domain.WebhookEventReleaseNew, payload.Event)
		assert.Equal(t, "1.0", payload.Version)
		assert.NotEmpty(t, payload.ID) // UUID should be present
		assert.NotNil(t, payload.Data)
		assert.NotNil(t, payload.Data.Release)

		// Assert release details
		assert.Equal(t, "Test.Release-Group", payload.Data.Release.Name)
		assert.Equal(t, "1080p", payload.Data.Release.Resolution)

		// Assert indexer details
		assert.NotNil(t, payload.Data.Indexer)
		assert.Equal(t, "MockIndexer", payload.Data.Indexer.Name)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	settings := &domain.Notification{
		Name:    "Test Webhook",
		Type:    domain.NotificationTypeWebhook,
		Webhook: server.URL,
		Enabled: true,
		Events:  []string{"RELEASE_NEW"},
	}

	log := logger.Mock().With().Logger()
	sender := NewWebhookSender(log, settings)

	payload := domain.NotificationPayload{
		Event:       domain.NotificationEventReleaseNew,
		Timestamp:   time.Now(),
		ReleaseName: "Test.Release-Group",
		Indexer:     "MockIndexer",
		Release: &domain.Release{
			TorrentName: "Test.Release-Group",
			Resolution:  "1080p",
			Indexer:     domain.IndexerMinimal{Identifier: "mock_indexer"},
		},
	}

	err := sender.Send(domain.NotificationEventReleaseNew, payload)
	assert.NoError(t, err)
}

func TestGenericWebhookSender_Send_Error(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	defer server.Close()

	settings := &domain.Notification{
		Name:    "Test Webhook",
		Type:    domain.NotificationTypeWebhook,
		Webhook: server.URL,
		Enabled: true,
	}

	log := logger.Mock().With().Logger()
	sender := NewWebhookSender(log, settings)

	err := sender.Send(domain.NotificationEventTest, domain.NotificationPayload{Event: domain.NotificationEventTest})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status: 400")
	assert.Contains(t, err.Error(), "bad request")
}

func TestGenericWebhookSender_CanSend(t *testing.T) {
	settings := &domain.Notification{
		Enabled: true,
		Type:    domain.NotificationTypeWebhook,
		Webhook: "http://localhost",
		Events:  []string{"RELEASE_NEW", "TEST"},
	}

	sender := NewWebhookSender(logger.Mock().With().Logger(), settings)

	assert.True(t, sender.CanSend(domain.NotificationEventReleaseNew))
	assert.True(t, sender.CanSend(domain.NotificationEventTest))
	assert.False(t, sender.CanSend(domain.NotificationEventPushApproved))

	settings.Enabled = false
	assert.False(t, sender.CanSend(domain.NotificationEventReleaseNew))
}

func TestGenericWebhookSender_Send_CustomMethod(t *testing.T) {
	// Test that a custom HTTP method is used when specified
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	settings := &domain.Notification{
		Name:    "Test Webhook with PUT",
		Type:    domain.NotificationTypeWebhook,
		Webhook: server.URL,
		Enabled: true,
		Events:  []string{"TEST"},
		Method:  "PUT",
	}

	log := logger.Mock().With().Logger()
	sender := NewWebhookSender(log, settings)

	err := sender.Send(domain.NotificationEventTest, domain.NotificationPayload{Event: domain.NotificationEventTest})
	assert.NoError(t, err)
}

func TestGenericWebhookSender_Send_CustomHeaders(t *testing.T) {
	// Test that custom headers are applied when specified
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
		// Default headers should still be set
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "autobrr", r.Header.Get("User-Agent"))
		// Event header should also be set (using namespaced value)
		assert.Equal(t, string(domain.WebhookEventTest), r.Header.Get("X-Autobrr-Event"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	settings := &domain.Notification{
		Name:    "Test Webhook with Headers",
		Type:    domain.NotificationTypeWebhook,
		Webhook: server.URL,
		Enabled: true,
		Events:  []string{"TEST"},
		Headers: "Authorization=Bearer test-token, X-Custom-Header=custom-value",
	}

	log := logger.Mock().With().Logger()
	sender := NewWebhookSender(log, settings)

	err := sender.Send(domain.NotificationEventTest, domain.NotificationPayload{Event: domain.NotificationEventTest})
	assert.NoError(t, err)
}
