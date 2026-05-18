// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"
	"time"

	"github.com/moistari/rls"
	"github.com/stretchr/testify/assert"
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
	assert.NotNil(t, result.Data)

	// Release Data
	assert.NotNil(t, result.Data.Release)
	assert.Equal(t, "Test.Release-Group", result.Data.Release.Name)
	assert.Equal(t, "Test Release", result.Data.Release.Title)
	assert.Equal(t, "1080p", result.Data.Release.Resolution)
	assert.Equal(t, uint64(1234567), result.Data.Release.Size)

	// Indexer Data
	assert.NotNil(t, result.Data.Indexer)
	assert.Equal(t, "MockIndexer", result.Data.Indexer.Name)
	assert.Equal(t, "mock_indexer", result.Data.Indexer.Identifier)

	// Filter Data
	assert.NotNil(t, result.Data.Filter)
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
	assert.NotNil(t, result.Data.Action)
	assert.Equal(t, "TestAction", result.Data.Action.Name)
	assert.Equal(t, "EXEC", result.Data.Action.Type)

	assert.NotNil(t, result.Data.Result)
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
