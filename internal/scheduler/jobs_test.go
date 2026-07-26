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

// pointTempDirAt makes os.TempDir resolve to dir so the job sweeps the test's
// directory rather than the real one. The assertion is not a formality: without
// it a change to os.TempDir would have the job deleting from the real temp dir.
func pointTempDirAt(t *testing.T, dir string) {
	t.Helper()

	t.Setenv("TMPDIR", dir) // unix
	t.Setenv("TMP", dir)    // windows
	t.Setenv("TEMP", dir)   // windows

	require.Equal(t, dir, os.TempDir(), "test needs os.TempDir to resolve to the test temp dir")
}

// The cleanup job used to resolve the current user from os.Getenv("UID"), which
// is a shell variable that is never exported to the process, then fall back to
// USER which is a name and not a numeric id. The ownership check parsed that as
// a uint and failed, so every file was skipped and nothing was ever removed.
func TestTempDirCleanupJob_RemovesStaleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	pointTempDirAt(t, tmpDir)

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

// Only regular files are ours to remove. A directory sharing the prefix could
// never be removed anyway and only produced an error line on every run.
func TestTempDirCleanupJob_LeavesDirectoriesAlone(t *testing.T) {
	tmpDir := t.TempDir()
	pointTempDirAt(t, tmpDir)

	dir := filepath.Join(tmpDir, "autobrr-dir")
	require.NoError(t, os.Mkdir(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "child"), []byte("x"), 0644))

	old := time.Now().Add(-25 * time.Hour)
	require.NoError(t, os.Chtimes(dir, old, old))

	NewTempDirCleanupJob(zerolog.Nop()).Run()

	assert.DirExists(t, dir, "a directory sharing the prefix must be left alone")
}
