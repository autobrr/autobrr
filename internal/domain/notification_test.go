// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"
	"time"

	"github.com/moistari/rls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebhookEvent(t *testing.T) {
	now := time.Now()
	payload := NotificationPayload{
		Event:          NotificationEventReleaseNew,
		Timestamp:      now,
		ReleaseName:    "Test.Release-Group",
		Indexer:        "MockIndexer",
		Protocol:       ReleaseProtocolTorrent,
		Implementation: ReleaseImplementationIRC,
		Filter:         "TestFilter",
		FilterID:       1,
	}

	release := &Release{
		Type:            rls.Movie,
		TorrentName:     "Test.Release-Group",
		Title:           "Test Release",
		Resolution:      "1080p",
		Source:          "WEB-DL",
		Codec:           []string{"H.264"},
		Size:            1234567,
		Seeders:         10,
		Leechers:        5,
		Freeleech:       true,
		MediaProcessing: "Encode",
		Indexer:         IndexerMinimal{Identifier: "mock_indexer"},
	}

	// set release on payload
	payload.Release = release

	id := "test-uuid-123"
	result := NewWebhookEvent(payload.Event, payload, id)

	assert.Equal(t, WebhookEventReleaseNew, result.Event)
	assert.Equal(t, id, result.ID)
	assert.Equal(t, now, result.Timestamp)
	assert.Equal(t, "1.0", result.Version)

	// Verify Data
	require.NotNil(t, result.Data)

	// Release Data
	require.NotNil(t, result.Data.Release)
	assert.Equal(t, "Test.Release-Group", result.Data.Release.Name)
	assert.Equal(t, "Test Release", result.Data.Release.Title)
	assert.Equal(t, "1080p", result.Data.Release.Resolution)
	assert.Equal(t, uint64(1234567), result.Data.Release.Size)

	// Indexer Data
	require.NotNil(t, result.Data.Indexer)
	assert.Equal(t, "MockIndexer", result.Data.Indexer.Name)
	assert.Equal(t, "mock_indexer", result.Data.Indexer.Identifier)

	// Filter Data
	require.NotNil(t, result.Data.Filter)
	assert.Equal(t, "TestFilter", result.Data.Filter.Name)
	assert.Equal(t, 1, result.Data.Filter.ID)

	// Action Data should be nil for this event type
	assert.Nil(t, result.Data.Action)
}

func TestNewWebhookEvent_Action(t *testing.T) {
	now := time.Now()
	payload := NotificationPayload{
		Event:        NotificationEventPushApproved,
		Timestamp:    now,
		Action:       "TestAction",
		ActionType:   ActionTypeExec,
		ActionClient: "qBittorrent",
		Status:       ReleasePushStatusApproved,
	}

	id := "test-uuid-456"
	result := NewWebhookEvent(payload.Event, payload, id)

	assert.Equal(t, WebhookEventActionApproved, result.Event)
	require.NotNil(t, result.Data.Action)
	assert.Equal(t, "TestAction", result.Data.Action.Name)
	assert.Equal(t, "EXEC", result.Data.Action.Type)

	require.NotNil(t, result.Data.Result)
	assert.Equal(t, "PUSH_APPROVED", result.Data.Result.Status)
}

func TestNewWebhookEvent_NilRelease(t *testing.T) {
	now := time.Now()
	payload := NotificationPayload{
		Event:     NotificationEventTest,
		Timestamp: now,
	}

	id := "test-uuid-789"
	result := NewWebhookEvent(payload.Event, payload, id)

	assert.Equal(t, WebhookEventTest, result.Event)
	assert.Equal(t, id, result.ID)
	assert.Nil(t, result.Data.Release)
}

func TestNotification_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		notification Notification
		wantErr      string
	}{
		{name: "notifiarr valid", notification: Notification{Type: NotificationTypeNotifiarr, APIKey: "0b6c2a3e-9f1d-4c8a-b7e2-5d4f3a2b1c0d"}},
		{name: "notifiarr missing api key", notification: Notification{Type: NotificationTypeNotifiarr}, wantErr: "missing notifiarr api key"},
		{name: "notifiarr short api key", notification: Notification{Type: NotificationTypeNotifiarr, APIKey: "abc123"}, wantErr: "notifiarr api key must be"},
		{name: "notifiarr api key with whitespace", notification: Notification{Type: NotificationTypeNotifiarr, APIKey: " 0b6c2a3e-9f1d-4c8a-b7e2-5d4f3a2b1c0"}, wantErr: "notifiarr api key must be"},
		{name: "notifiarr uppercase api key", notification: Notification{Type: NotificationTypeNotifiarr, APIKey: "0B6C2A3E-9F1D-4C8A-B7E2-5D4F3A2B1C0D"}, wantErr: "notifiarr api key must be"},
		{name: "discord missing webhook", notification: Notification{Type: NotificationTypeDiscord}, wantErr: "missing webhook url"},
		{name: "lunasea missing webhook", notification: Notification{Type: NotificationTypeLunaSea}, wantErr: "missing webhook url"},
		{name: "webhook missing url", notification: Notification{Type: NotificationTypeWebhook}, wantErr: "missing webhook url"},
		{name: "gotify missing token", notification: Notification{Type: NotificationTypeGotify, Host: "https://gotify.example.com"}, wantErr: "missing gotify application token"},
		{name: "ntfy missing host", notification: Notification{Type: NotificationTypeNtfy}, wantErr: "missing url"},
		{name: "shoutrrr missing host", notification: Notification{Type: NotificationTypeShoutrrr}, wantErr: "missing url"},
		{name: "pushover missing user key", notification: Notification{Type: NotificationTypePushover, APIKey: "token"}, wantErr: "missing pushover user key"},
		{name: "telegram missing chat id", notification: Notification{Type: NotificationTypeTelegram, Token: "token"}, wantErr: "missing telegram chat id"},
		{name: "telegram valid", notification: Notification{Type: NotificationTypeTelegram, Token: "token", Channel: "123"}},
		{name: "builtin", notification: Notification{Type: NotificationTypeBuiltin}},
		{name: "missing type", notification: Notification{}, wantErr: "unsupported notification type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.notification.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}

			assert.ErrorIs(t, err, ErrNotificationInvalid)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}
