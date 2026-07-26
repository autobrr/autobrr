// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package scheduler

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The cleanup job used to resolve the current user from os.Getenv("UID"), which
// is a shell variable that is never exported to the process, then fall back to
// USER which is a name and not a numeric id. The ownership check parsed that as
// a uint and failed, so every file was skipped and nothing was ever removed.
func TestTempDirCleanupJob_RemovesStaleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// os.TempDir reads these, which is how we point the job at the test dir
	t.Setenv("TMPDIR", tmpDir)
	t.Setenv("TMP", tmpDir)
	t.Setenv("TEMP", tmpDir)
	require.Equal(t, tmpDir, os.TempDir(), "test needs os.TempDir to resolve to the temp dir")

	stale := filepath.Join(tmpDir, "autobrr-stale")
	recent := filepath.Join(tmpDir, "autobrr-recent")
	unrelated := filepath.Join(tmpDir, "someone-elses-file")

	for _, name := range []string{stale, recent, unrelated} {
		require.NoError(t, os.WriteFile(name, []byte("torrent"), 0644))
	}

	// backdate everything that should be considered stale
	old := time.Now().Add(-25 * time.Hour)
	require.NoError(t, os.Chtimes(stale, old, old))
	require.NoError(t, os.Chtimes(unrelated, old, old))

	NewTempDirCleanupJob(zerolog.Nop()).Run()

	assert.NoFileExists(t, stale, "stale autobrr tmp file should be removed")
	assert.FileExists(t, recent, "recent autobrr tmp file should be kept")
	assert.FileExists(t, unrelated, "files without the autobrr prefix should be left alone")
}

// Shared seedboxes run several users off one system with a shared /tmp, so the
// job must only ever remove its own files.
func TestTempDirCleanupJob_LeavesOtherUsersFilesAlone(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("TMPDIR", tmpDir)
	t.Setenv("TMP", tmpDir)
	t.Setenv("TEMP", tmpDir)
	require.Equal(t, tmpDir, os.TempDir())

	old := time.Now().Add(-25 * time.Hour)

	// A symlink pointing at a stale file we do own. os.Stat follows it and
	// reports the target as ours and old, so the previous code would have
	// judged the target and unlinked the symlink.
	target := filepath.Join(t.TempDir(), "real-file")
	require.NoError(t, os.WriteFile(target, []byte("not a torrent"), 0644))
	require.NoError(t, os.Chtimes(target, old, old))

	link := filepath.Join(tmpDir, "autobrr-symlink")
	require.NoError(t, os.Symlink(target, link))

	// a directory sharing the prefix is not ours to remove either
	dir := filepath.Join(tmpDir, "autobrr-dir")
	require.NoError(t, os.Mkdir(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "child"), []byte("x"), 0644))
	require.NoError(t, os.Chtimes(dir, old, old))

	NewTempDirCleanupJob(zerolog.Nop()).Run()

	_, err := os.Lstat(link)
	assert.NoError(t, err, "a symlink must not be followed or removed")
	assert.FileExists(t, target, "the symlink target must be untouched")
	assert.DirExists(t, dir, "a directory sharing the prefix must be left alone")
}

func TestIsOwnedByCurrentUser(t *testing.T) {
	tmpDir := t.TempDir()

	path := filepath.Join(tmpDir, "autobrr-owned")
	require.NoError(t, os.WriteFile(path, []byte("torrent"), 0644))

	fileInfo, err := os.Stat(path)
	require.NoError(t, err)

	assert.True(t, isOwnedByCurrentUser(os.Geteuid(), fileInfo), "a file we just created is owned by us")
}

// The isolation that matters on a shared seedbox: another user's file must not
// be considered ours. /etc/passwd is root owned on every system we run on.
func TestIsOwnedByCurrentUser_OtherUser(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root, every file is ours")
	}

	fileInfo, err := os.Stat("/etc/passwd")
	require.NoError(t, err)

	assert.False(t, isOwnedByCurrentUser(os.Geteuid(), fileInfo), "a root owned file is not ours")
}
