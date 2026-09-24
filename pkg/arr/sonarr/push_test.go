// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package sonarr

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/autobrr/autobrr/pkg/arr"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Push_indexerNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/release/push", func(w http.ResponseWriter, r *http.Request) {
		payload, err := os.ReadFile("testdata/release_push_indexer_not_found.json")
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(payload)
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	client := New(Config{Hostname: ts.URL, APIKey: "mock-key", Log: zerolog.Nop()})

	rejections, err := client.Push(t.Context(), ReleasePushRequest{
		Title:            "That Show S01E01 1080p WEB-DL DDP5.1 H.264-GROUP",
		DownloadUrl:      "https://mock.local/download",
		Indexer:          "IPTorrents",
		Protocol:         "torrent",
		DownloadProtocol: "torrent",
	})

	assert.Nil(t, rejections)
	_, ok := errors.AsType[*arr.ErrorResponse](err)
	assert.True(t, ok)
	assert.EqualError(t, err, "Indexer with name 'IPTorrents' could not be found")
}
