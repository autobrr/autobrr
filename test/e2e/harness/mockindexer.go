// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

package harness

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// The mock indexer's ports are fixed: test/definitions/mock.yaml points autobrr
// at them, and the definition is also what a developer uses by hand. A test run
// therefore needs them free, which is the one thing stopping the e2e tests from
// running in parallel with a local mock indexer.
const (
	mockHTTPPort = 3999
	mockIRCPort  = 6697

	// MockTorrentID names the fixture the mock indexer serves. Announce lines
	// end in this id and the indexer resolves it to files/<id>.torrent.
	MockTorrentID = "1"

	// MockTorrentName is the release the announce advertises. It is scene-style
	// so autobrr's parser has something realistic to work with, and distinctive
	// enough for a filter to match it exactly.
	MockTorrentName = "Debian.Live.12.5.0.AMD64.Standard.ISO-E2E"
)

// MockIndexer is a running mock indexer: an IRC server that announces, plus an
// HTTP server that serves the announced .torrent.
type MockIndexer struct {
	t    testing.TB
	cmd  *exec.Cmd
	logs *syncBuffer
}

// StartMockIndexer boots the mock indexer with a torrent fixture in place and
// stops it when the test ends.
func StartMockIndexer(t testing.TB) *MockIndexer {
	t.Helper()

	// The mock indexer resolves files/ and feeds/ relative to its working
	// directory, so give it a directory of its own rather than polluting the
	// repo's own test/mockindexer/files.
	dir := t.TempDir()

	filesDir := filepath.Join(dir, "files")
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		t.Fatalf("harness: could not create mock indexer files dir: %v", err)
	}

	fixture := filepath.Join(filesDir, MockTorrentID+".torrent")
	if err := os.WriteFile(fixture, buildTorrent(MockTorrentName), 0o644); err != nil {
		t.Fatalf("harness: could not write torrent fixture: %v", err)
	}

	mock := &MockIndexer{t: t, logs: &syncBuffer{}}

	mock.cmd = exec.Command(mockIndexerBin, "--port", fmt.Sprint(mockHTTPPort))
	mock.cmd.Dir = dir
	mock.cmd.Stdout = mock.logs
	mock.cmd.Stderr = mock.logs

	if err := mock.cmd.Start(); err != nil {
		t.Fatalf("harness: could not start mock indexer: %v", err)
	}

	t.Cleanup(mock.stop)

	if err := waitForTCP(fmt.Sprintf("127.0.0.1:%d", mockIRCPort), 20*time.Second); err != nil {
		t.Fatalf("harness: mock indexer irc server did not come up: %v\n%s", err, mock.logs.String())
	}

	if err := waitForHTTP(mock.BaseURL()+"/", 20*time.Second); err != nil {
		t.Fatalf("harness: mock indexer http server did not come up: %v\n%s", err, mock.logs.String())
	}

	return mock
}

// BaseURL is where the mock indexer serves torrents. It has to match the urls
// entry in test/definitions/mock.yaml.
func (m *MockIndexer) BaseURL() string {
	return fmt.Sprintf("http://localhost:%d", mockHTTPPort)
}

// Announce posts a line to the mock indexer, which broadcasts it to the IRC
// channel autobrr is monitoring.
func (m *MockIndexer) Announce(line string) {
	m.t.Helper()

	resp, err := http.PostForm(m.BaseURL()+"/send", url.Values{"line": {line}})
	if err != nil {
		m.t.Fatalf("harness: could not post announce: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		m.t.Fatalf("harness: announce returned %d", resp.StatusCode)
	}
}

// AnnounceRelease posts an announce for name in the format mock.yaml parses.
func (m *MockIndexer) AnnounceRelease(name string) {
	m.t.Helper()

	m.Announce(fmt.Sprintf(
		"New Torrent Announcement: <PC :: Iso>  Name:'%s' uploaded by 'Anonymous' -  %s/torrent/%s",
		name, m.BaseURL(), MockTorrentID,
	))
}

func (m *MockIndexer) stop() {
	if m.cmd.Process == nil {
		return
	}

	_ = m.cmd.Process.Kill()
	_ = m.cmd.Wait()

	if m.t.Failed() {
		m.t.Logf("mock indexer logs:\n%s", m.logs.String())
	}
}

// waitForTCP polls addr until something accepts a connection.
func waitForTCP(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s: %w", addr, err)
		}

		time.Sleep(100 * time.Millisecond)
	}
}
