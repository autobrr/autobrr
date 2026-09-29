// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package radarr

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	leakAPIKey    = "0000000000000000000000000apikey"
	leakPasskey   = "1111111111111111111111111passkey"
	leakBasicAuth = "2222222222222222222222222basicauth"
)

// These strings end up on the release as a rejection, which is stored in
// release_action_status.rejections, rendered in the web UI and forwarded to
// notification providers.
func TestClient_TransportErrorDoesNotLeakCredentials(t *testing.T) {
	t.Parallel()

	// a closed server gives a deterministic transport-level failure
	srv := httptest.NewServer(nil)
	closed := srv.URL
	srv.Close()

	u, err := url.Parse(closed)
	require.NoError(t, err)
	// userinfo in the hostname is accepted and reaches url.String(), so it has
	// to be redacted the way net/http already redacts it in *url.Error
	u.User = url.UserPassword("admin", leakBasicAuth)

	c := New(Config{Hostname: u.String(), APIKey: leakAPIKey, Log: zerolog.Nop()})

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "get",
			call: func() error { _, err := c.Test(t.Context()); return err },
		},
		{
			name: "getJSON",
			call: func() error { _, err := c.GetTags(t.Context()); return err },
		},
		{
			name: "postBody",
			call: func() error {
				_, err := c.Push(t.Context(), ReleasePushRequest{
					Title:       "Movie 2024 1080p BluRay x264-GROUP",
					DownloadUrl: "https://tracker.example/download/1?torrent_pass=" + leakPasskey,
					Protocol:    "torrent",
					Indexer:     "mock",
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.call()

			require.Error(t, err)
			for _, secret := range []string{leakAPIKey, leakPasskey, leakBasicAuth} {
				assert.NotContains(t, err.Error(), secret)
			}
		})
	}
}
