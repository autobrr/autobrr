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
		assert.Equal(t, string(domain.NotificationEventReleaseNew), r.Header.Get("X-Autobrr-Event"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload domain.GenericWebhookPayload
		err = json.Unmarshal(body, &payload)
		require.NoError(t, err)

		assert.Equal(t, domain.NotificationEventReleaseNew, payload.Event)
		assert.Equal(t, "Test.Release-Group", payload.ReleaseName)
		assert.Equal(t, "MockIndexer", payload.Indexer)
		assert.Equal(t, "1080p", payload.Resolution)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	settings := &domain.Notification{
		Name:    "Test Webhook",
		Type:    domain.NotificationTypeGenericWebhook,
		Webhook: server.URL,
		Enabled: true,
		Events:  []string{"RELEASE_NEW"},
	}

	log := logger.Mock().With().Logger()
	sender := NewGenericWebhookSender(log, settings)

	payload := domain.NotificationPayload{
		Event:       domain.NotificationEventReleaseNew,
		Timestamp:   time.Now(),
		ReleaseName: "Test.Release-Group",
		Indexer:     "MockIndexer",
		Release: &domain.Release{
			TorrentName: "Test.Release-Group",
			Resolution:  "1080p",
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
		Type:    domain.NotificationTypeGenericWebhook,
		Webhook: server.URL,
		Enabled: true,
	}

	log := logger.Mock().With().Logger()
	sender := NewGenericWebhookSender(log, settings)

	err := sender.Send(domain.NotificationEventTest, domain.NotificationPayload{Event: domain.NotificationEventTest})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status: 400")
	assert.Contains(t, err.Error(), "bad request")
}

func TestGenericWebhookSender_CanSend(t *testing.T) {
	settings := &domain.Notification{
		Enabled: true,
		Type:    domain.NotificationTypeGenericWebhook,
		Webhook: "http://localhost",
		Events:  []string{"RELEASE_NEW", "TEST"},
	}

	sender := NewGenericWebhookSender(logger.Mock().With().Logger(), settings)

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
		Type:    domain.NotificationTypeGenericWebhook,
		Webhook: server.URL,
		Enabled: true,
		Events:  []string{"TEST"},
		Method:  "PUT",
	}

	log := logger.Mock().With().Logger()
	sender := NewGenericWebhookSender(log, settings)

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
		// Event header should also be set
		assert.Equal(t, string(domain.NotificationEventTest), r.Header.Get("X-Autobrr-Event"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	settings := &domain.Notification{
		Name:    "Test Webhook with Headers",
		Type:    domain.NotificationTypeGenericWebhook,
		Webhook: server.URL,
		Enabled: true,
		Events:  []string{"TEST"},
		Headers: "Authorization=Bearer test-token, X-Custom-Header=custom-value",
	}

	log := logger.Mock().With().Logger()
	sender := NewGenericWebhookSender(log, settings)

	err := sender.Send(domain.NotificationEventTest, domain.NotificationPayload{Event: domain.NotificationEventTest})
	assert.NoError(t, err)
}
