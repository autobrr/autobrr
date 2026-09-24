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
