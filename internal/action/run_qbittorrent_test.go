// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/downloader"

	"github.com/autobrr/go-qbittorrent"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeQbittorrent answers the qBittorrent WebAPI calls runQbittorrent makes. Like qBittorrent
// 5.2 it answers a torrent it already holds with 409 Conflict. Failing endpoints answer 500,
// except a failing add, which answers 415 like an invalid torrent file.
type fakeQbittorrent struct {
	mu       sync.Mutex
	torrents map[string]bool
	failing  map[string]bool
	calls    []string
}

func newFakeQbittorrent(t *testing.T, torrents []string, failing ...string) (*fakeQbittorrent, *httptest.Server) {
	t.Helper()

	q := &fakeQbittorrent{torrents: map[string]bool{}, failing: map[string]bool{}}
	for _, hash := range torrents {
		q.torrents[hash] = true
	}
	for _, endpoint := range failing {
		q.failing[endpoint] = true
	}

	srv := httptest.NewServer(http.HandlerFunc(q.handle))
	t.Cleanup(srv.Close)

	return q, srv
}

func (q *fakeQbittorrent) handle(w http.ResponseWriter, r *http.Request) {
	endpoint := strings.TrimPrefix(r.URL.Path, "/api/v2/")

	q.mu.Lock()
	defer q.mu.Unlock()

	q.calls = append(q.calls, endpoint)

	if q.failing[endpoint] {
		// go-qbittorrent maps 415 to the same error as the 409 for a duplicate
		if endpoint == "torrents/add" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch endpoint {
	case "torrents/add":
		if q.torrents[testTorrentHash] {
			w.WriteHeader(http.StatusConflict)
			return
		}

		q.torrents[testTorrentHash] = true
		_, _ = w.Write([]byte("Ok."))

	case "torrents/info":
		_ = r.ParseForm()

		var torrents []qbittorrent.Torrent
		for hash := range strings.SplitSeq(r.Form.Get("hashes"), "|") {
			if q.torrents[hash] {
				torrents = append(torrents, qbittorrent.Torrent{Hash: hash})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(torrents)

	case "torrents/trackers":
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]qbittorrent.TorrentTracker{{Url: "https://tracker.example/announce", Status: qbittorrent.TrackerStatusNotWorking}})

	case "app/preferences":
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"queueing_enabled":true}`))

	default:
		w.WriteHeader(http.StatusOK)
	}
}

func (q *fakeQbittorrent) called(endpoint string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, c := range q.calls {
		if c == endpoint {
			return true
		}
	}

	return false
}

const testTorrentHash = "abc123"

func newQbittorrentTestService(host string) *Service {
	client := qbittorrent.NewClient(qbittorrent.Config{Host: host, APIKey: "key"})
	cfg := &domain.Downloader{ID: 1, Name: "qbittorrent", Type: domain.DownloaderTypeQbittorrent, Enabled: true, Host: host}

	return &Service{
		log:            zerolog.Nop(),
		downloaderSvc:  &fakeDownloaderService{instance: downloader.NewInstance(cfg, client)},
		rlsDownloadSvc: &fakeDownloadService{},
	}
}

func TestService_runQbittorrent_results(t *testing.T) {
	tests := []struct {
		name           string
		inClient       []string
		failing        []string
		action         domain.Action
		wantErr        bool
		wantRejections []string
	}{
		{
			name:     "torrent already in client",
			inClient: []string{testTorrentHash},
			action:   domain.Action{ReAnnounceSkip: true},
		},
		{
			name:     "torrent already in client but lookup fails",
			inClient: []string{testTorrentHash},
			failing:  []string{"torrents/info"},
			action:   domain.Action{ReAnnounceSkip: true},
			wantErr:  true,
		},
		{
			name:    "invalid torrent file",
			failing: []string{"torrents/add"},
			action:  domain.Action{ReAnnounceSkip: true},
			wantErr: true,
		},
		{
			name:    "priority fails after add",
			failing: []string{"torrents/topPrio"},
			action:  domain.Action{ReAnnounceSkip: true, PriorityLayout: domain.PriorityLayoutMax},
		},
		{
			name:    "re-announce fails after add",
			failing: []string{"torrents/trackers"},
			action:  domain.Action{ReAnnounceInterval: 1, ReAnnounceMaxAttempts: 1},
		},
		{
			name:           "re-announce took too long with delete",
			action:         domain.Action{ReAnnounceInterval: 1, ReAnnounceMaxAttempts: 1, ReAnnounceDelete: true},
			wantRejections: []string{"re-announce took too long for hash: abc123"},
		},
		{
			name:    "delete fails after re-announce took too long",
			failing: []string{"torrents/delete"},
			action:  domain.Action{ReAnnounceInterval: 1, ReAnnounceMaxAttempts: 1, ReAnnounceDelete: true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, srv := newFakeQbittorrent(t, tt.inClient, tt.failing...)
			s := newQbittorrentTestService(srv.URL)

			act := tt.action
			act.Name = "qbittorrent"
			act.Type = domain.ActionTypeQbittorrent
			act.ClientID = 1

			rls := domain.NewRelease(domain.IndexerMinimal{Identifier: "mock"})
			rls.TorrentName = "That.Movie.2023.1080p.BluRay.x264-GROUP"
			rls.Protocol = domain.ReleaseProtocolTorrent
			rls.DownloadURL = "https://tracker.example/download/1"

			rejections, err := s.runQbittorrent(zerolog.Nop().WithContext(t.Context()), &act, rls)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantRejections, rejections)
			assert.True(t, q.called("torrents/add"))
			if tt.action.PriorityLayout == domain.PriorityLayoutMax {
				assert.True(t, q.called("torrents/topPrio"))
			}
		})
	}
}
