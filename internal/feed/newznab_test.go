// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
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

type mockNewznabFeedClient struct {
	limit      int
	pages      map[int][]newznab.FeedItem
	errOffsets map[int]error
	offsets    []int
}

func (m *mockNewznabFeedClient) WithHTTPClient(_ *http.Client) {}

func (m *mockNewznabFeedClient) Search(_ context.Context, _ string, _ []int, offset int) (*newznab.SearchResponse, error) {
	m.offsets = append(m.offsets, offset)

	if err := m.errOffsets[offset]; err != nil {
		return nil, err
	}

	items := m.pages[offset]

	respItems := make([]*newznab.FeedItem, 0, len(items))
	for i := range items {
		respItems = append(respItems, &items[i])
	}

	return &newznab.SearchResponse{Items: respItems, Limit: m.limit}, nil
}

func TestNewznabJob_getFeed_pagination(t *testing.T) {
	newItem := func(guid string) newznab.FeedItem {
		return newznab.FeedItem{GUID: guid, Title: "Title." + guid}
	}

	guids := func(items []newznab.FeedItem) []string {
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, item.GUID)
		}
		return out
	}

	newJob := func(client newznabClient, cache *stubFeedCacheRepo, lastRun time.Time) *NewznabJob {
		return &NewznabJob{
			Log: zerolog.New(io.Discard),
			Feed: &domain.Feed{
				Indexer:  domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed"},
				LastRun:  lastRun,
				Settings: &domain.FeedSettingsJSON{MaxPages: domain.MaxFeedPages},
			},
			Client:    client,
			Repo:      &mockFeedRepo{},
			CacheRepo: cache,
		}
	}

	recentRun := time.Now().Add(-time.Hour)

	twoFullPages := func() map[int][]newznab.FeedItem {
		return map[int][]newznab.FeedItem{
			0: {newItem("new-1"), newItem("new-2"), newItem("new-3")},
			3: {newItem("new-4"), newItem("new-5"), newItem("new-6")},
			6: {newItem("new-7")},
		}
	}

	t.Run("stops at the cached boundary on page 2", func(t *testing.T) {
		client := &mockNewznabFeedClient{
			limit: 3,
			pages: map[int][]newznab.FeedItem{
				0: {newItem("new-1"), newItem("new-2"), newItem("new-3")},
				3: {newItem("new-4"), newItem("old-1"), newItem("old-2")},
			},
		}
		cache := &stubFeedCacheRepo{existing: map[string]bool{"old-1": true, "old-2": true}}

		items, err := newJob(client, cache, recentRun).getFeed(t.Context())
		require.NoError(t, err)

		// oldest first: the second page's items precede the first page's
		assert.Equal(t, []string{"new-4", "new-3", "new-2", "new-1"}, guids(items))
		assert.Equal(t, []int{0, 3}, client.offsets)
	})

	t.Run("cache write failure drops the page and stops paginating", func(t *testing.T) {
		client := &mockNewznabFeedClient{limit: 3, pages: twoFullPages()}
		cache := &stubFeedCacheRepo{existing: map[string]bool{}, putErr: errors.New("disk full")}

		items, err := newJob(client, cache, recentRun).getFeed(t.Context())
		require.NoError(t, err)

		assert.Empty(t, items)
		assert.Equal(t, []int{0}, client.offsets)
	})

	t.Run("fetch error on page 2 keeps the cached first page", func(t *testing.T) {
		client := &mockNewznabFeedClient{
			limit:      3,
			pages:      twoFullPages(),
			errOffsets: map[int]error{3: errors.New("429 too many requests")},
		}

		items, err := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, recentRun).getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-3", "new-2", "new-1"}, guids(items))
		assert.Equal(t, []int{0, 3}, client.offsets)
	})

	t.Run("first run fetches only the first page", func(t *testing.T) {
		client := &mockNewznabFeedClient{limit: 3, pages: twoFullPages()}

		items, err := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, time.Time{}).getFeed(t.Context())
		require.NoError(t, err)

		assert.Len(t, items, 3)
		assert.Equal(t, []int{0}, client.offsets)
	})

	t.Run("last run older than the cache ttl fetches only the first page", func(t *testing.T) {
		client := &mockNewznabFeedClient{limit: 3, pages: twoFullPages()}

		_, err := newJob(client, &stubFeedCacheRepo{existing: map[string]bool{}}, time.Now().AddDate(0, 0, -40)).getFeed(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []int{0}, client.offsets)
	})
}
