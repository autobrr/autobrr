//go:build !windows

// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package scheduler

// These cover the ownership rules the cleanup job relies on, which are POSIX
// concepts. The Windows build of isOwnedByCurrentUser answers true for anything
// it can stat, and the temp dir there is per user, so none of this applies.

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Shared seedboxes run several users off one system with a shared /tmp, so
// another user can leave a symlink named like one of our temp files. os.Stat
// follows it and reports the target's owner and mtime, which they control.
func TestTempDirCleanupJob_DoesNotFollowSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	pointTempDirAt(t, tmpDir)

	old := time.Now().Add(-25 * time.Hour)

	// the target is stale and genuinely ours, so judging it rather than the
	// link is what made the previous code unlink the symlink
	target := filepath.Join(t.TempDir(), "real-file")
	require.NoError(t, os.WriteFile(target, []byte("not a torrent"), 0644))
	require.NoError(t, os.Chtimes(target, old, old))

	link := filepath.Join(tmpDir, "autobrr-symlink")
	require.NoError(t, os.Symlink(target, link))

	NewTempDirCleanupJob(zerolog.Nop()).Run()

	_, err := os.Lstat(link)
	assert.NoError(t, err, "a symlink must not be followed or removed")
	assert.FileExists(t, target, "the symlink target must be untouched")
}

func TestIsOwnedByCurrentUser(t *testing.T) {
	path := filepath.Join(t.TempDir(), "autobrr-owned")
	require.NoError(t, os.WriteFile(path, []byte("torrent"), 0644))

	fileInfo, err := os.Stat(path)
	require.NoError(t, err)

	assert.True(t, isOwnedByCurrentUser(os.Geteuid(), fileInfo), "a file we just created is owned by us")
}

// The isolation that matters on a shared seedbox: another user's file must not
// be considered ours. /etc/passwd is root owned on any system we run on.
func TestIsOwnedByCurrentUser_OtherUser(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root, every file is ours")
	}

	fileInfo, err := os.Stat("/etc/passwd")
	require.NoError(t, err)

	assert.False(t, isOwnedByCurrentUser(os.Geteuid(), fileInfo), "a root owned file is not ours")
}
