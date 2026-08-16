// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

package harness

import (
	"bytes"
	"testing"

	"github.com/autobrr/go-torrent/metainfo"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The announce test is only meaningful if autobrr accepts the torrent the mock
// indexer serves. Parse the fixture with the same library autobrr uses so a
// malformed fixture fails here, with a clear reason, rather than surfacing as a
// download error three layers down.
func TestBuildTorrentIsValid(t *testing.T) {
	raw := buildTorrent(MockTorrentName)

	mi, err := metainfo.Load(bytes.NewReader(raw))
	require.NoError(t, err, "fixture must be loadable as a torrent")

	info, err := mi.UnmarshalInfo()
	require.NoError(t, err, "fixture must have a parseable info dictionary")

	assert.Equal(t, MockTorrentName, info.Name)
	assert.NotZero(t, info.Length)
	assert.Len(t, info.Pieces, metainfo.HashSize, "pieces must be a whole number of piece hashes")
	assert.NotEmpty(t, mi.HashInfoBytes().String(), "fixture must yield an infohash")
}
