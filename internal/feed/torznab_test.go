package feed

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTorznabSource_toRelease(t *testing.T) {
	const (
		magnetURI  = "magnet:?xt=urn:btih:deadbeef"
		proxyURL   = "http://jackett:9117/dl/mock/?jackett_apikey=key&file=Some.Release"
		torrentURL = "https://fake-feed.com/download/00000.torrent"
	)

	torrentEnclosure := &torznab.Enclosure{URL: torrentURL, Type: "application/x-bittorrent"}
	magnetEnclosure := &torznab.Enclosure{URL: magnetURI, Type: "application/x-bittorrent"}

	tests := []struct {
		name            string
		settings        *domain.FeedSettingsJSON
		item            torznab.FeedItem
		wantDownloadURL string
		wantMagnetURI   string
	}{
		{
			name:            "no feed settings keeps the torrent url",
			item:            torznab.FeedItem{Link: torrentURL, Enclosure: torrentEnclosure},
			wantDownloadURL: torrentURL,
		},
		{
			name:            "torrent type ignores the magneturl attr",
			settings:        &domain.FeedSettingsJSON{DownloadType: domain.FeedDownloadTypeTorrent},
			item:            torznab.FeedItem{Link: torrentURL, Enclosure: torrentEnclosure, MagnetURI: magnetURI},
			wantDownloadURL: torrentURL,
		},
		{
			name:          "magnet in the link",
			settings:      &domain.FeedSettingsJSON{DownloadType: domain.FeedDownloadTypeMagnet},
			item:          torznab.FeedItem{Link: magnetURI},
			wantMagnetURI: magnetURI,
		},
		{
			name:          "magnet in the enclosure",
			settings:      &domain.FeedSettingsJSON{DownloadType: domain.FeedDownloadTypeMagnet},
			item:          torznab.FeedItem{Link: proxyURL, Enclosure: magnetEnclosure},
			wantMagnetURI: magnetURI,
		},
		{
			name:            "magneturl attr wins over the proxy link",
			settings:        &domain.FeedSettingsJSON{DownloadType: domain.FeedDownloadTypeMagnet},
			item:            torznab.FeedItem{Link: proxyURL, MagnetURI: magnetURI},
			wantDownloadURL: proxyURL,
			wantMagnetURI:   magnetURI,
		},
		{
			name:            "proxy link is kept for ResolveMagnetURI to follow",
			settings:        &domain.FeedSettingsJSON{DownloadType: domain.FeedDownloadTypeMagnet},
			item:            torznab.FeedItem{Link: proxyURL},
			wantDownloadURL: proxyURL,
			wantMagnetURI:   proxyURL,
		},
		{
			name:          "magnet never stays in the download url",
			settings:      &domain.FeedSettingsJSON{DownloadType: domain.FeedDownloadTypeTorrent},
			item:          torznab.FeedItem{Link: proxyURL, Enclosure: magnetEnclosure},
			wantMagnetURI: magnetURI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &torznabSource{
				log: zerolog.Nop(),
				feed: &domain.Feed{
					Indexer:  domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed"},
					Settings: tt.settings,
				},
			}

			tt.item.Title = "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP"

			rls := src.toRelease(&tt.item)

			assert.Equal(t, tt.wantDownloadURL, rls.DownloadURL, "download url")
			assert.Equal(t, tt.wantMagnetURI, rls.MagnetURI, "magnet uri")
		})
	}
}

func TestTorznabSource_fetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixture := "testdata/torznab/torznab_response.xml"
		if r.URL.Query().Get("t") == "caps" {
			fixture = "testdata/torznab/caps_response.xml"
		}

		payload, err := os.ReadFile(fixture)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	f := &domain.Feed{
		Type:    string(domain.FeedTypeTorznab),
		URL:     srv.URL,
		Indexer: domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed"},
	}

	src, err := (&Service{}).newSource(t.Context(), f, zerolog.Nop())
	require.NoError(t, err)

	res, err := src.fetch(t.Context())
	require.NoError(t, err)

	assert.NotEmpty(t, res.raw)
	require.NotEmpty(t, res.entries)

	for _, e := range res.entries {
		assert.NotEmpty(t, e.key)

		rls := e.release()
		assert.Equal(t, e.title, rls.TorrentName)
		assert.Equal(t, domain.ReleaseImplementationTorznab, rls.Implementation)
	}
}
