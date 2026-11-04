// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/downloader"

	"github.com/hekmon/transmissionrpc/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTransmissionServer answers the Transmission RPC calls used by runTransmission
// with a single tracker reporting the given seeder count, and records every method it receives.
func newTransmissionServer(t *testing.T, seeders int64) (*httptest.Server, func() []string) {
	t.Helper()

	var (
		m       sync.Mutex
		methods []string
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Tag    int    `json:"tag"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))

		m.Lock()
		methods = append(methods, req.Method)
		m.Unlock()

		var args any = map[string]any{}

		switch req.Method {
		case "torrent-add":
			args = map[string]any{"torrent-added": map[string]any{"id": 1, "hashString": "0123456789abcdef0123456789abcdef01234567", "name": "test"}}
		case "torrent-get":
			args = map[string]any{"torrents": []any{map[string]any{"trackerStats": []any{map[string]any{"seederCount": seeders, "leecherCount": 0}}}}}
		}

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"arguments": args, "result": "success", "tag": req.Tag}))
	}))
	t.Cleanup(ts.Close)

	return ts, func() []string {
		m.Lock()
		defer m.Unlock()

		return methods
	}
}

func newTransmissionTestService(t *testing.T, host string) *Service {
	t.Helper()

	endpoint, err := url.Parse(host)
	require.NoError(t, err)

	client, err := transmissionrpc.New(endpoint, nil)
	require.NoError(t, err)

	cfg := &domain.Downloader{ID: 1, Name: "transmission", Type: domain.DownloaderTypeTransmission, Enabled: true, Host: host}

	return &Service{
		log:           zerolog.Nop(),
		downloaderSvc: &fakeDownloaderService{instance: downloader.NewInstance(cfg, client)},
	}
}

func countMethod(methods []string, method string) int {
	n := 0
	for _, m := range methods {
		if m == method {
			n++
		}
	}

	return n
}

func TestService_runTransmission_reannounce(t *testing.T) {
	tests := []struct {
		name            string
		seeders         int64
		deleteStalled   bool
		wantRejected    bool
		wantReannounces int
		wantRemoved     bool
	}{
		{name: "stalled_delete", seeders: 0, deleteStalled: true, wantRejected: true, wantReannounces: 2, wantRemoved: true},
		{name: "stalled_keep", seeders: 0, deleteStalled: false, wantRejected: false, wantReannounces: 2, wantRemoved: false},
		{name: "working_tracker", seeders: 5, deleteStalled: true, wantRejected: false, wantReannounces: 0, wantRemoved: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, methods := newTransmissionServer(t, tt.seeders)
			svc := newTransmissionTestService(t, ts.URL)

			action := &domain.Action{
				Name:                  "transmission",
				Type:                  domain.ActionTypeTransmission,
				ClientID:              1,
				ReAnnounceInterval:    1,
				ReAnnounceMaxAttempts: 2,
				ReAnnounceDelete:      tt.deleteStalled,
			}
			release := &domain.Release{TorrentName: "test", MagnetURI: "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567"}

			rejections, err := svc.runTransmission(context.Background(), action, release)
			require.NoError(t, err)

			if tt.wantRejected {
				if assert.Len(t, rejections, 1) {
					assert.Contains(t, rejections[0], "reannounce took too long")
				}
			} else {
				assert.Empty(t, rejections)
			}

			got := methods()
			assert.Equal(t, tt.wantReannounces, countMethod(got, "torrent-reannounce"), "torrent-reannounce calls: %v", got)
			assert.Equal(t, tt.wantRemoved, countMethod(got, "torrent-remove") == 1, "torrent-remove calls: %v", got)
		})
	}
}
