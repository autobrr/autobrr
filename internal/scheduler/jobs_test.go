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

func TestIsOwnedByCurrentUser(t *testing.T) {
	tmpDir := t.TempDir()

	path := filepath.Join(tmpDir, "autobrr-owned")
	require.NoError(t, os.WriteFile(path, []byte("torrent"), 0644))

	fileInfo, err := os.Stat(path)
	require.NoError(t, err)

	assert.True(t, isOwnedByCurrentUser(os.Getuid(), fileInfo), "a file we just created is owned by us")
}
