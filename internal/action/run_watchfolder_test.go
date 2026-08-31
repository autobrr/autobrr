// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testInfoHash = "9c8f7e6d5a4b3c2d1e0f9a8b7c6d5e4f3a2b1c0d"

func testTorrentRelease() *domain.Release {
	return &domain.Release{
		TorrentName:         "Some.Movie.2024.1080p-GRP",
		TorrentHash:         testInfoHash,
		Protocol:            domain.ReleaseProtocolTorrent,
		TorrentDataRawBytes: []byte("d8:announce0:4:infod4:name4:teste"),
	}
}

// The torrent is written straight from memory now, so the watch folder never
// sees a tmp file and the name comes from the infohash.
func TestWatchFolder_WritesTorrentNamedAfterInfoHash(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	release := testTorrentRelease()

	err := (&Service{}).runWatchFolder(t.Context(), &domain.Action{WatchFolder: dir}, release)
	require.NoError(t, err)

	want := filepath.Join(dir, "autobrr-"+testInfoHash+".torrent")
	require.FileExists(t, want)

	contents, err := os.ReadFile(want)
	require.NoError(t, err)
	assert.Equal(t, release.TorrentDataRawBytes, contents, "the torrent must be written verbatim")
}

// A watch folder configured with an explicit .torrent path keeps using it.
func TestWatchFolder_HonoursExplicitFileName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "chosen-name.torrent")

	err := (&Service{}).runWatchFolder(t.Context(), &domain.Action{WatchFolder: target}, testTorrentRelease())
	require.NoError(t, err)

	assert.FileExists(t, target)
}

func TestWatchFolder_RejectsMagnet(t *testing.T) {
	t.Parallel()

	release := testTorrentRelease()
	release.MagnetURI = "magnet:?xt=urn:btih:" + testInfoHash

	err := (&Service{}).runWatchFolder(t.Context(), &domain.Action{WatchFolder: t.TempDir()}, release)

	assert.Error(t, err, "watch folder cannot write a magnet link")
}

// Without the torrent in memory there is nothing to write, and we should say so
// rather than leave an empty file behind for a client to choke on.
func TestWatchFolder_RejectsMissingTorrent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	release := testTorrentRelease()
	release.TorrentDataRawBytes = nil

	err := (&Service{}).runWatchFolder(t.Context(), &domain.Action{WatchFolder: dir}, release)
	require.Error(t, err)

	entries, readErr := os.ReadDir(dir)
	require.NoError(t, readErr)
	assert.Empty(t, entries, "no file should be created when the torrent is missing")
}
