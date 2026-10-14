// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFeed_PaginationMaxPages(t *testing.T) {
	tests := []struct {
		name     string
		settings *FeedSettingsJSON
		want     int
	}{
		{
			name: "no settings falls back to the default",
			want: DefaultFeedPages,
		},
		{
			name:     "configured value is used",
			settings: &FeedSettingsJSON{MaxPages: 3},
			want:     3,
		},
		{
			name:     "zero falls back to the default",
			settings: &FeedSettingsJSON{MaxPages: 0},
			want:     DefaultFeedPages,
		},
		{
			name:     "the cap itself is allowed",
			settings: &FeedSettingsJSON{MaxPages: MaxFeedPages},
			want:     MaxFeedPages,
		},
		{
			name:     "values above the cap are clamped",
			settings: &FeedSettingsJSON{MaxPages: 9999},
			want:     MaxFeedPages,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Feed{Settings: tt.settings}

			assert.Equal(t, tt.want, f.PaginationMaxPages())
		})
	}
}

func TestFeed_CacheTTLDays(t *testing.T) {
	tests := []struct {
		name     string
		settings *FeedSettingsJSON
		want     int
	}{
		{name: "no settings falls back to the default", want: DefaultFeedCacheTTLDays},
		{name: "configured value is used", settings: &FeedSettingsJSON{CacheTTLDays: 7}, want: 7},
		{name: "zero falls back to the default", settings: &FeedSettingsJSON{CacheTTLDays: 0}, want: DefaultFeedCacheTTLDays},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Feed{Settings: tt.settings}.CacheTTLDays())
		})
	}
}

func TestFeed_CacheCoversLastRun(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		lastRun  time.Time
		settings *FeedSettingsJSON
		want     bool
	}{
		{name: "never run", want: false},
		{name: "1 day ago with the default ttl", lastRun: now.AddDate(0, 0, -1), want: true},
		{name: "32 days ago with the default ttl", lastRun: now.AddDate(0, 0, -32), want: false},
		{name: "5 days ago with a 3 day ttl", lastRun: now.AddDate(0, 0, -5), settings: &FeedSettingsJSON{CacheTTLDays: 3}, want: false},
		{name: "2 days ago with a 3 day ttl", lastRun: now.AddDate(0, 0, -2), settings: &FeedSettingsJSON{CacheTTLDays: 3}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Feed{LastRun: tt.lastRun, Settings: tt.settings}

			assert.Equal(t, tt.want, f.CacheCoversLastRun(now))
		})
	}
}
