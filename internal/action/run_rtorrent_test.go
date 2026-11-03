// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/downloader"

	"github.com/autobrr/go-rtorrent"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rtorrentCall struct {
	Method string   `xml:"methodName"`
	Params []string `xml:"params>param>value>string"`
}

// newRTorrentServer answers XML-RPC like an rTorrent with the ruTorrent ratio
// plugin views and records every call it receives.
func newRTorrentServer(t *testing.T) (*httptest.Server, func() []rtorrentCall) {
	t.Helper()

	var (
		m     sync.Mutex
		calls []rtorrentCall
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var call rtorrentCall
		require.NoError(t, xml.NewDecoder(r.Body).Decode(&call))

		m.Lock()
		calls = append(calls, call)
		m.Unlock()

		w.Header().Set("Content-Type", "text/xml")

		switch call.Method {
		case "view.list":
			w.Write([]byte(`<?xml version="1.0"?><methodResponse><params><param><value><array><data><value><string>main</string></value><value><string>rat_0</string></value><value><string>rat_1</string></value></data></array></value></param></params></methodResponse>`))
		default:
			w.Write([]byte(`<?xml version="1.0"?><methodResponse><params><param><value><i8>0</i8></value></param></params></methodResponse>`))
		}
	}))
	t.Cleanup(ts.Close)

	return ts, func() []rtorrentCall {
		m.Lock()
		defer m.Unlock()

		return calls
	}
}

func newRTorrentTestService(host string) *Service {
	cfg := &domain.Downloader{ID: 1, Name: "rtorrent", Type: domain.DownloaderTypeRTorrent, Enabled: true, Host: host}
	client := rtorrent.NewClient(rtorrent.Config{Addr: host})

	return &Service{
		log:           zerolog.Nop(),
		downloaderSvc: &fakeDownloaderService{instance: downloader.NewInstance(cfg, client)},
	}
}

func TestService_runRTorrent_ratioGroupAndPriority(t *testing.T) {
	tests := []struct {
		name      string
		action    domain.Action
		wantCalls []rtorrentCall
		wantErr   string
	}{
		{
			name:   "none",
			action: domain.Action{ClientID: 1},
			wantCalls: []rtorrentCall{
				{Method: "load.start", Params: []string{""}},
			},
		},
		{
			name:   "priority_high",
			action: domain.Action{ClientID: 1, PriorityLayout: domain.PriorityLayoutHigh},
			wantCalls: []rtorrentCall{
				{Method: "load.start", Params: []string{"", `d.priority.set="3"`}},
			},
		},
		{
			name:   "qbittorrent_priority_ignored",
			action: domain.Action{ClientID: 1, PriorityLayout: domain.PriorityLayoutMax},
			wantCalls: []rtorrentCall{
				{Method: "load.start", Params: []string{""}},
			},
		},
		{
			name:   "ratio_group",
			action: domain.Action{ClientID: 1, Label: "tv", RatioGroup: "rat_1"},
			wantCalls: []rtorrentCall{
				{Method: "view.list"},
				{Method: "load.start", Params: []string{"", `d.custom1.set="tv"`, `view.set_visible="rat_1"`}},
			},
		},
		{
			name:      "ratio_group_missing",
			action:    domain.Action{ClientID: 1, RatioGroup: "rat_5"},
			wantCalls: []rtorrentCall{{Method: "view.list"}},
			wantErr:   "could not find ratio group 'rat_5' on client: rtorrent, check that the ruTorrent ratio plugin is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, calls := newRTorrentServer(t)
			s := newRTorrentTestService(ts.URL)

			release := &domain.Release{TorrentName: "Test.Release-GROUP", MagnetURI: "magnet:?xt=urn:btih:3f9aac158c7de8dfcab171ea58a17aabdf7fbc93"}

			_, err := s.runRTorrent(t.Context(), &tt.action, release)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantCalls, calls())
		})
	}
}
