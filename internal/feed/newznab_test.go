// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"io"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/newznab"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewznabJob_processItems_metadata(t *testing.T) {
	pubDate := time.Date(2026, time.September, 24, 5, 58, 24, 0, time.UTC)

	j := &NewznabJob{
		Log: zerolog.New(io.Discard),
		Feed: &domain.Feed{
			Indexer: domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed"},
		},
	}

	tests := []struct {
		name        string
		item        newznab.FeedItem
		wantTVDB    int
		wantPubDate time.Time
	}{
		{
			name: "ids and publish date are mapped",
			item: newznab.FeedItem{
				TvdbId:  "12345",
				PubDate: newznab.Time{Time: pubDate},
			},
			wantTVDB:    12345,
			wantPubDate: pubDate,
		},
		{
			name: "non numeric id and missing publish date are left zero",
			item: newznab.FeedItem{TvdbId: "abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.item.Title = "That.Show.S01E01.1080p.WEB-DL.H.264-GROUP"

			releases, err := j.processItems([]newznab.FeedItem{tt.item})
			require.NoError(t, err)
			require.Len(t, releases, 1)

			assert.Equal(t, tt.wantTVDB, releases[0].MetaTVDB, "tvdb id")
			assert.Equal(t, tt.wantPubDate, releases[0].PublishDate, "publish date")
		})
	}
}
