// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

package e2e_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/autobrr/autobrr/test/e2e/harness"

	"github.com/stretchr/testify/require"
)

// The whole point of autobrr, end to end: an announce appears on IRC, a filter
// matches it, and an action does something with the torrent.
//
// Everything up to the announce is configured through the UI, because that is
// how a user sets this up and the parts most likely to break are the forms. The
// tail of the test then leaves the browser behind: the announce goes in over the
// mock indexer's HTTP API and the assertion is a file on disk, which is the real
// output of the pipeline rather than a rendering of it.
func TestAnnounceMatchesFilterAndRunsAction(t *testing.T) {
	mock := harness.StartMockIndexer(t)

	app := harness.Start(t)
	page := app.NewAuthedPage()

	watchDir := t.TempDir()

	// The mock indexer definition needs an RSS key and an IRC nick before
	// autobrr will consider the indexer configured.
	page.Open("/settings/indexers")
	page.AddNew()
	page.Select("MockIndexer")
	page.Fill("settings.rsskey", "0123456789abcdef0123")
	page.Fill("irc.nick", "autobrr-e2e")
	page.Submit()
	page.ExpectRow("MockIndexer")

	// Adding the indexer creates its IRC network, disabled. Enabling it is what
	// makes autobrr connect to the mock indexer's server.
	page.Open("/settings/irc")
	page.ExpectRow("Mock")
	page.Toggle("enabled")

	// A filter only ever sees releases from indexers it is attached to, and only
	// while both are enabled.
	page.Open("/filters")
	page.ClickButton("Add new")
	page.Fill("name", "E2E Announce")
	page.Submit()

	page.SelectMulti("indexers", "MockIndexer")

	// New filters are created disabled, and FindByIndexerIdentifier only
	// considers enabled ones.
	page.Toggle("enabled")

	page.ClickLink("Advanced")
	page.Expand("Release Names", "match_releases")
	page.Fill("match_releases", harness.MockTorrentName)

	page.ClickLink("Actions")
	page.ClickButton("Add new")
	page.SelectListbox("Watch dir")
	page.Fill("actions[0].watch_folder", watchDir)

	page.Submit()
	page.ExpectText("E2E Announce was updated successfully")

	// Announcing before autobrr has joined the channel would send the line into
	// an empty room and fail much later, for reasons that look nothing like the
	// cause.
	app.WaitForChannelMonitoring(harness.AnnounceChannel, 30*time.Second)

	mock.AnnounceRelease(harness.MockTorrentName)

	torrent := waitForTorrent(t, watchDir, 30*time.Second)
	require.NotEmpty(t, torrent, "autobrr should have written the torrent to the watch folder")

	contents, err := os.ReadFile(torrent)
	require.NoError(t, err)
	require.Contains(t, string(contents), harness.MockTorrentName,
		"the watch folder should hold the torrent that was announced")

	t.Logf("action wrote %s", filepath.Base(torrent))
}

// waitForTorrent polls dir until a .torrent shows up, returning its path.
func waitForTorrent(t *testing.T, dir string, timeout time.Duration) string {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		matches, err := filepath.Glob(filepath.Join(dir, "*.torrent"))
		require.NoError(t, err)

		if len(matches) > 0 {
			return matches[0]
		}

		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for a torrent in %s", dir)

	return ""
}
