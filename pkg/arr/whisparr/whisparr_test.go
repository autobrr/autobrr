// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package whisparr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAPIKey = "mock-key"

// newTestServer serves the v2 endpoints (series) or the v3 ones (movie),
// answering 404 for the other version like the real apps do.
func newTestServer(t *testing.T, version int) *httptest.Server {
	t.Helper()

	statusFile := "testdata/system_status_v2_response.json"
	if version == VersionV3 {
		statusFile = "testdata/system_status_v3_response.json"
	}

	mux := http.NewServeMux()

	serve := func(file string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Api-Key") != testAPIKey {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			payload, err := os.ReadFile(file)
			require.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(payload)
		}
	}

	mux.HandleFunc("/api/v3/system/status", serve(statusFile))
	mux.HandleFunc("/api/v3/release/push", serve("testdata/release_push_response.json"))

	if version == VersionV3 {
		mux.HandleFunc("/api/v3/movie", serve("testdata/movie_response.json"))
	} else {
		mux.HandleFunc("/api/v3/series", serve("testdata/series_response.json"))
	}

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	return ts
}

func newTestClient(hostname string, version int) *Client {
	return New(Config{
		Hostname: hostname,
		APIKey:   testAPIKey,
		Version:  version,
		Log:      zerolog.Nop(),
	})
}

func TestSystemStatusResponse_MajorVersion(t *testing.T) {
	tests := []struct {
		version string
		want    int
	}{
		{version: "2.2.0.108", want: VersionV2},
		{version: "3.3.7.979", want: VersionV3},
		{version: "3", want: VersionV3},
		{version: "", want: 0},
		{version: "nightly", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			assert.Equal(t, tt.want, SystemStatusResponse{Version: tt.version}.MajorVersion())
		})
	}
}

func TestClient_Test(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	t.Run("v2 client against v2 server", func(t *testing.T) {
		ts := newTestServer(t, VersionV2)

		status, err := newTestClient(ts.URL, VersionV2).Test(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "2.2.0.108", status.Version)
	})

	t.Run("v3 client against v3 server", func(t *testing.T) {
		ts := newTestServer(t, VersionV3)

		status, err := newTestClient(ts.URL, VersionV3).Test(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "3.3.7.979", status.Version)
	})

	t.Run("v2 client against v3 server reports the mismatch", func(t *testing.T) {
		ts := newTestServer(t, VersionV3)

		_, err := newTestClient(ts.URL, VersionV2).Test(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configured for Whisparr v2")
		assert.Contains(t, err.Error(), "3.3.7.979")
	})

	t.Run("v3 client against v2 server reports the mismatch", func(t *testing.T) {
		ts := newTestServer(t, VersionV2)

		_, err := newTestClient(ts.URL, VersionV3).Test(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configured for Whisparr v3")
	})

	t.Run("bad api key", func(t *testing.T) {
		ts := newTestServer(t, VersionV2)

		client := New(Config{Hostname: ts.URL, APIKey: "wrong", Version: VersionV2, Log: zerolog.Nop()})

		_, err := client.Test(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})
}

func TestClient_GetAllSeries(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	ts := newTestServer(t, VersionV2)

	series, err := newTestClient(ts.URL, VersionV2).GetAllSeries(context.Background())
	require.NoError(t, err)
	require.Len(t, series, 2)

	assert.Equal(t, "Brazzers", series[0].Title)
	assert.True(t, series[0].Monitored)
	assert.Equal(t, []int{1}, series[0].Tags)

	assert.Equal(t, "Tushy", series[1].Title)
	assert.False(t, series[1].Monitored)
	assert.Empty(t, series[1].Tags)
}

func TestClient_GetMovies(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	ts := newTestServer(t, VersionV3)

	movies, err := newTestClient(ts.URL, VersionV3).GetMovies(context.Background())
	require.NoError(t, err)
	require.Len(t, movies, 2)

	assert.Equal(t, "Brazzers Goes Black", movies[0].Title)
	assert.Equal(t, "movie", movies[0].ItemType)
	assert.True(t, movies[0].Monitored)
	require.Len(t, movies[0].AlternateTitles, 1)
	assert.Equal(t, "Brazzers Goes Black 1", movies[0].AlternateTitles[0].Title)

	assert.Equal(t, "Brazzers Beach", movies[1].Title)
	assert.Equal(t, "scene", movies[1].ItemType)
	assert.False(t, movies[1].Monitored)
}

// The item endpoints do not overlap between versions, a v3 instance has no
// series and a v2 instance has no movies.
func TestClient_ItemEndpointsAreVersionSpecific(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	t.Run("no movie endpoint on v2", func(t *testing.T) {
		ts := newTestServer(t, VersionV2)

		_, err := newTestClient(ts.URL, VersionV2).GetMovies(context.Background())
		require.Error(t, err)
	})

	t.Run("no series endpoint on v3", func(t *testing.T) {
		ts := newTestServer(t, VersionV3)

		_, err := newTestClient(ts.URL, VersionV3).GetAllSeries(context.Background())
		require.Error(t, err)
	})
}

func TestClient_Push(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	ts := newTestServer(t, VersionV3)

	rejections, err := newTestClient(ts.URL, VersionV3).Push(context.Background(), ReleasePushRequest{
		Title:            "Brazzers.Goes.Black.2018.1080p.BluRay.x264-GROUP",
		DownloadUrl:      ts.URL + "/download",
		Size:             1073741824,
		Indexer:          "autobrr",
		Protocol:         "torrent",
		DownloadProtocol: "torrent",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"Unknown Movie. Unable to match to existing movie in Library using release title."}, rejections)
}

// A temporarily rejected release comes back with rejected false and temporarilyRejected
// true, and waits in Whisparr's pending queue instead of being grabbed, so its rejections
// still have to reach the caller.
func TestClient_Push_temporarilyRejected(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/release/push", func(w http.ResponseWriter, r *http.Request) {
		payload, err := os.ReadFile("testdata/release_push_temporarily_rejected_response.json")
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(payload)
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	rejections, err := newTestClient(ts.URL, VersionV3).Push(context.Background(), ReleasePushRequest{
		Title:            "Brazzers.Goes.Black.2018.1080p.BluRay.x264-GROUP",
		DownloadUrl:      ts.URL + "/download",
		Size:             1073741824,
		Indexer:          "autobrr",
		Protocol:         "torrent",
		DownloadProtocol: "torrent",
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"Waiting for a better quality release"}, rejections)
}
