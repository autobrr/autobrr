// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/downloader"
	"github.com/autobrr/autobrr/pkg/arr/sonarr"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_arrDownloadClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		settings domain.DownloaderSettings
		action   domain.Action
		wantID   int
		wantName string
	}{
		{
			name:     "client_settings",
			settings: domain.DownloaderSettings{ExternalDownloadClientId: 1, ExternalDownloadClient: "qbit"},
			wantID:   1,
			wantName: "qbit",
		},
		{
			name:     "action_name_replaces_client_id",
			settings: domain.DownloaderSettings{ExternalDownloadClientId: 1},
			action:   domain.Action{ExternalDownloadClient: "qbit-2"},
			wantName: "qbit-2",
		},
		{
			name:     "action_id_replaces_client_name",
			settings: domain.DownloaderSettings{ExternalDownloadClient: "qbit"},
			action:   domain.Action{ExternalDownloadClientID: 2},
			wantID:   2,
		},
		{
			name:     "action_pair",
			settings: domain.DownloaderSettings{ExternalDownloadClientId: 1, ExternalDownloadClient: "qbit"},
			action:   domain.Action{ExternalDownloadClientID: 2, ExternalDownloadClient: "qbit-2"},
			wantID:   2,
			wantName: "qbit-2",
		},
		{
			name: "none",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, name := arrDownloadClient(tt.settings, &tt.action)
			assert.Equal(t, tt.wantID, id)
			assert.Equal(t, tt.wantName, name)
		})
	}
}

func TestArrPublishDate(t *testing.T) {
	t.Parallel()

	pubDate := time.Date(2026, time.September, 24, 5, 58, 24, 0, time.UTC)

	t.Run("uses the feed publish date when set", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "2026-09-24T05:58:24Z", arrPublishDate(&domain.Release{PublishDate: pubDate}))
	})

	t.Run("formats a non utc feed date in utc", func(t *testing.T) {
		t.Parallel()

		local := pubDate.In(time.FixedZone("BST", 60*60))

		assert.Equal(t, "2026-09-24T05:58:24Z", arrPublishDate(&domain.Release{PublishDate: local}))
	})

	t.Run("preserves dates after the feed placeholder cutoff", func(t *testing.T) {
		t.Parallel()

		date := time.Date(1970, time.April, 1, 0, 0, 1, 0, time.UTC)

		assert.Equal(t, "1970-04-01T00:00:01Z", arrPublishDate(&domain.Release{PublishDate: date}))
	})

	fallback := []struct {
		name        string
		publishDate time.Time
	}{
		{name: "falls back to now when the feed publish date is zero"},
		{name: "falls back to now when the feed publish date is before the epoch", publishDate: time.Date(1969, time.December, 31, 0, 0, 0, 0, time.UTC)},
		{name: "falls back to now when the feed publish date is the epoch", publishDate: time.Unix(0, 0)},
		{name: "falls back to now when the feed publish date is at the placeholder cutoff", publishDate: time.Date(1970, time.April, 1, 0, 0, 0, 0, time.UTC)},
		{name: "falls back to now when the feed publish date is in the future", publishDate: time.Now().Add(2 * time.Hour)},
	}

	for _, tt := range fallback {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := time.Parse(time.RFC3339, arrPublishDate(&domain.Release{PublishDate: tt.publishDate}))
			require.NoError(t, err)

			assert.WithinDuration(t, time.Now(), got, 5*time.Second)
		})
	}
}

type fakeDownloaderService struct {
	downloaderService
	instance *downloader.Instance
}

func (f *fakeDownloaderService) GetInstance(_ context.Context, _ int32) (*downloader.Instance, error) {
	return f.instance, nil
}

// newSonarrPushServer answers release/push like Sonarr v5, refusing any indexer name it does not know.
func newSonarrPushServer(t *testing.T, knownIndexer string) (*httptest.Server, func() []string) {
	t.Helper()

	var (
		m        sync.Mutex
		indexers []string
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/release/push", func(w http.ResponseWriter, r *http.Request) {
		var req sonarr.ReleasePushRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		m.Lock()
		indexers = append(indexers, req.Indexer)
		m.Unlock()

		w.Header().Set("Content-Type", "application/json")

		if req.Indexer != "" && req.Indexer != knownIndexer {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"message":"Indexer with name '` + req.Indexer + `' could not be found"}`))
			return
		}

		w.Write([]byte(`[{"title":"` + req.Title + `","approved":true,"rejected":false,"rejections":[]}]`))
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	return ts, func() []string {
		m.Lock()
		defer m.Unlock()

		return indexers
	}
}

func newSonarrTestService(host string) *Service {
	cfg := &domain.Downloader{ID: 1, Name: "sonarr", Type: domain.DownloaderTypeSonarr, Enabled: true, Host: host}
	client := sonarr.New(sonarr.Config{Hostname: host, APIKey: "mock-key", Log: zerolog.Nop()})

	return &Service{
		log:           zerolog.Nop(),
		downloaderSvc: &fakeDownloaderService{instance: downloader.NewInstance(cfg, client)},
	}
}

func newSonarrTestRelease() *domain.Release {
	return &domain.Release{
		TorrentName: "WeCrashed.S01E01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD",
		DownloadURL: "https://mock.local/download",
		Protocol:    domain.ReleaseProtocolTorrent,
		Indexer:     domain.IndexerMinimal{ID: 0, Name: "IPTorrents", Identifier: "ipt", IdentifierExternal: "IPTorrents"},
	}
}

func TestService_runSonarr_indexerNotFound(t *testing.T) {
	ts, pushed := newSonarrPushServer(t, "IPTorrents (Prowlarr)")
	s := newSonarrTestService(ts.URL)

	rejections, err := s.runSonarr(t.Context(), &domain.Action{ClientID: 1}, newSonarrTestRelease())
	assert.EqualError(t, err, "sonarr: failed to push release: WeCrashed.S01E01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD: Indexer with name 'IPTorrents' could not be found")
	assert.Nil(t, rejections)
	assert.Equal(t, []string{"IPTorrents"}, pushed())
}

func TestService_runSonarr_indexerFound(t *testing.T) {
	ts, pushed := newSonarrPushServer(t, "IPTorrents")
	s := newSonarrTestService(ts.URL)

	rejections, err := s.runSonarr(t.Context(), &domain.Action{ClientID: 1}, newSonarrTestRelease())
	require.NoError(t, err)
	assert.Nil(t, rejections)
	assert.Equal(t, []string{"IPTorrents"}, pushed())
}

// newSonarrCaptureServer accepts every release/push and returns the raw JSON body of the last one,
// so a test can tell an omitted field from a zero value.
func newSonarrCaptureServer(t *testing.T) (*httptest.Server, func() map[string]any) {
	t.Helper()

	var (
		m      sync.Mutex
		pushed map[string]any
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/release/push", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		m.Lock()
		pushed = body
		m.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"approved":true,"rejected":false,"rejections":[]}]`))
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	return ts, func() map[string]any {
		m.Lock()
		defer m.Unlock()

		return pushed
	}
}

func TestService_runSonarr_pushesFeedMetadata(t *testing.T) {
	ts, pushed := newSonarrCaptureServer(t)

	release := newSonarrTestRelease()
	release.MetaTVDB = 12345
	release.PublishDate = time.Date(2026, time.September, 24, 5, 58, 24, 0, time.UTC)

	_, err := newSonarrTestService(ts.URL).runSonarr(t.Context(), &domain.Action{ClientID: 1}, release)
	require.NoError(t, err)

	assert.EqualValues(t, 12345, pushed()["tvdbId"])
	assert.Equal(t, "2026-09-24T05:58:24Z", pushed()["publishDate"])
}

func TestService_runSonarr_omitsMissingTvdbID(t *testing.T) {
	ts, pushed := newSonarrCaptureServer(t)

	_, err := newSonarrTestService(ts.URL).runSonarr(t.Context(), &domain.Action{ClientID: 1}, newSonarrTestRelease())
	require.NoError(t, err)

	assert.NotContains(t, pushed(), "tvdbId")
}
