package feed

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestTorznabJob_processItems(t *testing.T) {
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
			j := &TorznabJob{
				Log: zerolog.New(io.Discard),
				Feed: &domain.Feed{
					Indexer:  domain.IndexerMinimal{Name: "Mock Feed", Identifier: "mock-feed"},
					Settings: tt.settings,
				},
			}

			tt.item.Title = "Some.Release.Title.2022.09.22.720p.WEB.h264-GROUP"

			releases, err := j.processItems([]torznab.FeedItem{tt.item})
			assert.NoError(t, err)
			assert.Len(t, releases, 1)

			assert.Equal(t, tt.wantDownloadURL, releases[0].DownloadURL, "download url")
			assert.Equal(t, tt.wantMagnetURI, releases[0].MagnetURI, "magnet uri")
		})
	}
}

func TestTorznabJob_RunE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryType := r.URL.Query().Get("t")
		switch queryType {
		case "search":
			payload, err := os.ReadFile("testdata/torznab/torznab_response.xml")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			w.Write(payload)
			break

		case "caps":
			payload, err := os.ReadFile("testdata/torznab/caps_response.xml")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			w.Write(payload)
			break
		}
	}))
	defer srv.Close()

	type fields struct {
		Feed       *domain.Feed
		Name       string
		Log        zerolog.Logger
		URL        string
		Client     *torznab.Client
		Repo       jobFeedRepo
		CacheRepo  jobFeedCacheRepo
		ReleaseSvc jobReleaseSvc
		attempts   int
		errors     []error
		JobID      int
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "test",
			fields: fields{
				Name: "test",
				Log:  zerolog.New(io.Discard),
				Feed: &domain.Feed{
					MaxAge: 0,
					Indexer: domain.IndexerMinimal{
						ID:                 0,
						Name:               "Mock Feed",
						Identifier:         "mock-feed",
						IdentifierExternal: "Mock Indexer",
					},
				},
				URL:        srv.URL,
				Client:     torznab.NewClient(torznab.Config{Host: srv.URL}),
				Repo:       &mockFeedRepo{},
				CacheRepo:  &mockFeedCacheRepo{},
				ReleaseSvc: &mockReleaseSvc{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &TorznabJob{
				Feed:       tt.fields.Feed,
				Name:       tt.fields.Name,
				Log:        tt.fields.Log,
				URL:        tt.fields.URL,
				Client:     tt.fields.Client,
				Repo:       tt.fields.Repo,
				CacheRepo:  tt.fields.CacheRepo,
				ReleaseSvc: tt.fields.ReleaseSvc,
				attempts:   tt.fields.attempts,
				errors:     tt.fields.errors,
				JobID:      tt.fields.JobID,
			}
			err := j.RunE(t.Context())
			assert.NoError(t, err)
		})
	}
}
