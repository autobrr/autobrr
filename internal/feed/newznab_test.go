// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/newznab"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewznabSource_toRelease(t *testing.T) {
	const (
		title  = "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP"
		guid   = "https://indexer.local/details/abc"
		nzbURL = "https://indexer.local/getnzb/abc.nzb"
	)

	tests := []struct {
		name            string
		item            newznab.FeedItem
		wantDownloadURL string
		wantSize        uint64
		wantCategory    string
		wantCategories  []string
		wantMetaTMDB    int
	}{
		{
			name:            "nzb enclosure is the download url",
			item:            newznab.FeedItem{Size: 100, Enclosure: &newznab.Enclosure{Url: nzbURL, Length: 200, Type: "application/x-nzb"}},
			wantDownloadURL: nzbURL,
			wantSize:        100,
		},
		{
			name:            "enclosure length fills a missing size",
			item:            newznab.FeedItem{Enclosure: &newznab.Enclosure{Url: nzbURL, Length: 200, Type: "application/x-nzb"}},
			wantDownloadURL: nzbURL,
			wantSize:        200,
		},
		{
			name: "non nzb enclosure is ignored",
			item: newznab.FeedItem{Enclosure: &newznab.Enclosure{Url: "https://indexer.local/x.torrent", Type: "application/x-bittorrent"}},
		},
		{
			name:           "single category is set as the category",
			item:           newznab.FeedItem{Categories: newznab.Categories{{ID: 5040, Name: "TV/HD"}}},
			wantCategory:   "TV/HD",
			wantCategories: []string{"TV/HD", "5040"},
		},
		{
			name:           "several categories only fill the list",
			item:           newznab.FeedItem{Categories: newznab.Categories{{ID: 5000, Name: "TV"}, {ID: 5040, Name: "TV/HD"}}},
			wantCategories: []string{"TV", "5000", "TV/HD", "5040"},
		},
		{
			name:         "tmdb id is parsed",
			item:         newznab.FeedItem{TmdbId: "1234"},
			wantMetaTMDB: 1234,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &newznabSource{
				log:  zerolog.Nop(),
				feed: &domain.Feed{Indexer: domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed"}},
			}

			tt.item.Title = title
			tt.item.GUID = guid

			rls := src.toRelease(&tt.item)

			assert.Equal(t, domain.ReleaseImplementationNewznab, rls.Implementation)
			assert.Equal(t, domain.ReleaseProtocolNzb, rls.Protocol)
			assert.Equal(t, title, rls.TorrentName)
			assert.Equal(t, guid, rls.InfoURL)
			assert.Equal(t, tt.wantDownloadURL, rls.DownloadURL, "download url")
			assert.Equal(t, tt.wantSize, rls.Size, "size")
			assert.Equal(t, tt.wantCategory, rls.Category, "category")
			assert.ElementsMatch(t, tt.wantCategories, rls.Categories, "categories")
			assert.Equal(t, tt.wantMetaTMDB, rls.MetaTMDB, "tmdb")
		})
	}
}

func TestNewznabSource_fetch(t *testing.T) {
	const response = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
<title>Mock Indexer</title>
<item>
<title>Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP</title>
<guid>https://indexer.local/details/abc</guid>
<pubDate>Mon, 05 Oct 2026 10:00:00 +0000</pubDate>
<enclosure url="https://indexer.local/getnzb/abc.nzb" length="1073741824" type="application/x-nzb"/>
</item>
<item>
<title>Item.Without.Guid</title>
</item>
</channel>
</rss>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")

		if r.URL.Query().Get("t") == "caps" {
			_, _ = w.Write([]byte(`<caps/>`))
			return
		}

		_, _ = w.Write([]byte(response))
	}))
	defer srv.Close()

	f := &domain.Feed{
		Type:    string(domain.FeedTypeNewznab),
		URL:     srv.URL,
		Indexer: domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed"},
	}

	src, err := (&Service{}).newSource(t.Context(), f, zerolog.Nop())
	require.NoError(t, err)

	res, err := src.fetch(t.Context())
	require.NoError(t, err)

	assert.Contains(t, res.raw, "Mock Indexer")
	require.Len(t, res.entries, 1, "the item without a guid is skipped")

	e := res.entries[0]
	assert.Equal(t, "https://indexer.local/details/abc", e.key)
	assert.Equal(t, time.Date(2026, time.October, 5, 10, 0, 0, 0, time.UTC), e.pubDate.UTC())

	rls := e.release()
	assert.Equal(t, "https://indexer.local/getnzb/abc.nzb", rls.DownloadURL)
}
