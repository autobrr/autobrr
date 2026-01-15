// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewGenericWebhookPayload(t *testing.T) {
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
	}

	result := NewGenericWebhookPayload(payload, release)

	assert.Equal(t, NotificationEventReleaseNew, result.Event)
	assert.Equal(t, now, result.Timestamp)
	assert.Equal(t, "Test.Release-Group", result.ReleaseName)
	assert.Equal(t, "Test.Release-Group", result.TorrentName)
	assert.Equal(t, "Test Release", result.Title)
	assert.Equal(t, "1080p", result.Resolution)
	assert.Equal(t, "WEB-DL", result.Source)
	assert.Equal(t, []string{"H.264"}, result.Codec)
	assert.Equal(t, uint64(1234567), result.Size)
	assert.Equal(t, 10, result.Seeders)
	assert.Equal(t, 5, result.Leechers)
	assert.True(t, result.Freeleech)
	assert.Equal(t, "Encode", result.MediaProcessing)
	assert.Equal(t, "TestFilter", result.Filter)
	assert.Equal(t, 1, result.FilterID)
}

func TestNewGenericWebhookPayload_NilRelease(t *testing.T) {
	now := time.Now()
	payload := NotificationPayload{
		Event:     NotificationEventTest,
		Timestamp: now,
	}

	result := NewGenericWebhookPayload(payload, nil)

	assert.Equal(t, NotificationEventTest, result.Event)
	assert.Equal(t, now, result.Timestamp)
	assert.Empty(t, result.TorrentName)
	assert.Empty(t, result.Title)
}
