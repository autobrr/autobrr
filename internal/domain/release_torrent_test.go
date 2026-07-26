// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelease_TorrentReader(t *testing.T) {
	t.Parallel()

	r := &Release{TorrentDataRawBytes: []byte("torrent contents")}

	got, err := io.ReadAll(r.TorrentReader())
	require.NoError(t, err)
	assert.Equal(t, []byte("torrent contents"), got)

	// the reader must not consume the release, later actions need the bytes too
	again, err := io.ReadAll(r.TorrentReader())
	require.NoError(t, err)
	assert.Equal(t, []byte("torrent contents"), again)
}

func TestRelease_WriteTemporaryFile(t *testing.T) {
	t.Parallel()

	r := &Release{TorrentName: "Test.Release-GRP", TorrentDataRawBytes: []byte("torrent contents")}

	require.NoError(t, r.WriteTemporaryFile())
	require.NotEmpty(t, r.TorrentTmpFile)

	t.Cleanup(func() {
		_ = os.Remove(r.TorrentTmpFile)
	})

	contents, err := os.ReadFile(r.TorrentTmpFile)
	require.NoError(t, err)
	assert.Equal(t, []byte("torrent contents"), contents)

	// writing again is a no-op so several actions share the one file
	first := r.TorrentTmpFile
	require.NoError(t, r.WriteTemporaryFile())
	assert.Equal(t, first, r.TorrentTmpFile)
}

func TestRelease_WriteTemporaryFileWithoutContents(t *testing.T) {
	t.Parallel()

	r := &Release{TorrentName: "Test.Release-GRP"}

	assert.Error(t, r.WriteTemporaryFile(), "writing a tmp file without the torrent should fail loudly")
	assert.Empty(t, r.TorrentTmpFile)
}

func TestRelease_CleanupTemporaryFiles(t *testing.T) {
	t.Parallel()

	r := &Release{TorrentName: "Test.Release-GRP", TorrentDataRawBytes: []byte("torrent contents")}
	require.NoError(t, r.WriteTemporaryFile())

	tmpFile := r.TorrentTmpFile
	require.FileExists(t, tmpFile)

	require.NoError(t, r.CleanupTemporaryFiles())

	assert.NoFileExists(t, tmpFile, "tmp file should be removed")
	assert.Empty(t, r.TorrentTmpFile)
	assert.Empty(t, r.TorrentDataRawBytes, "torrent should not be held after processing")
}

func TestAction_NeedsTorrentDownloaded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		actionType ActionType
		want       bool
	}{
		{ActionTypeQbittorrent, true},
		{ActionTypeDelugeV1, true},
		{ActionTypeDelugeV2, true},
		{ActionTypeRTorrent, true},
		{ActionTypeTransmission, true},
		{ActionTypePorla, true},
		{ActionTypeWatchFolder, true},
		{ActionTypeRadarr, false},
		{ActionTypeSonarr, false},
		{ActionTypeSabnzbd, false},
		{ActionTypeExec, false},
		{ActionTypeWebhook, false},
		{ActionTypeTest, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.actionType), func(t *testing.T) {
			a := &Action{Type: tt.actionType}
			assert.Equal(t, tt.want, a.NeedsTorrentDownloaded())
		})
	}
}

func TestAction_CheckMacrosNeedTorrent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		action    Action
		wantBytes bool
		wantFile  bool
	}{
		{
			name:      "path macro in exec args needs both",
			action:    Action{Type: ActionTypeExec, ExecArgs: "--file {{.TorrentPathName}}"},
			wantBytes: true,
			wantFile:  true,
		},
		{
			name:      "raw bytes macro needs contents only",
			action:    Action{Type: ActionTypeExec, ExecArgs: "--data {{.TorrentDataRawBytes}}"},
			wantBytes: true,
			wantFile:  false,
		},
		{
			name:      "hash macro needs contents only",
			action:    Action{Type: ActionTypeWebhook, WebhookData: `{"hash":"{{.TorrentHash}}"}`},
			wantBytes: true,
			wantFile:  false,
		},
		{
			name:      "path macro in watch folder is covered",
			action:    Action{Type: ActionTypeExec, WatchFolder: "/watch/{{.TorrentPathName}}"},
			wantBytes: true,
			wantFile:  true,
		},
		{
			name:      "path macro in save path is covered",
			action:    Action{Type: ActionTypeExec, SavePath: "/data/{{.TorrentPathName}}"},
			wantBytes: true,
			wantFile:  true,
		},
		{
			name:      "unrelated macros need nothing",
			action:    Action{Type: ActionTypeExec, ExecArgs: "--name {{.TorrentName}}", SavePath: "/data/{{.Indexer}}"},
			wantBytes: false,
			wantFile:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release := &Release{TorrentName: "Test.Release-GRP"}

			assert.Equal(t, tt.wantBytes, tt.action.CheckMacrosNeedRawDataBytes(release), "raw data bytes")
			assert.Equal(t, tt.wantFile, tt.action.CheckMacrosNeedTorrentTmpFile(release), "tmp file")
		})
	}
}

// An exec script handed the path may consume or move the file, and cleanup
// should treat that as done rather than as an error.
func TestRelease_CleanupTemporaryFilesAlreadyGone(t *testing.T) {
	t.Parallel()

	r := &Release{TorrentName: "Test.Release-GRP", TorrentDataRawBytes: []byte("torrent contents")}
	require.NoError(t, r.WriteTemporaryFile())

	require.NoError(t, os.Remove(r.TorrentTmpFile))

	assert.NoError(t, r.CleanupTemporaryFiles(), "an already removed tmp file is not an error")
	assert.Empty(t, r.TorrentTmpFile)
	assert.Empty(t, r.TorrentDataRawBytes)
}

func TestFilterExternal_NeedTorrent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		external   FilterExternal
		wantBytes  bool
		wantTmpFil bool
	}{
		{
			// this used to be checked on WebhookData only, so an exec filter
			// asking for the contents never triggered a download
			name:      "raw bytes in exec args",
			external:  FilterExternal{ExecArgs: "--data {{.TorrentDataRawBytes}}"},
			wantBytes: true,
		},
		{
			name:      "raw bytes in webhook data",
			external:  FilterExternal{WebhookData: `{"data":"{{.TorrentDataRawBytes}}"}`},
			wantBytes: true,
		},
		{
			name:       "path macro needs the file on disk",
			external:   FilterExternal{ExecArgs: "--file {{.TorrentPathName}}"},
			wantBytes:  true,
			wantTmpFil: true,
		},
		{
			// exposed as a macro but was checked nowhere
			name:       "tmp file macro needs the file on disk",
			external:   FilterExternal{WebhookData: `{"path":"{{.TorrentTmpFile}}"}`},
			wantBytes:  true,
			wantTmpFil: true,
		},
		{
			name:      "hash needs the contents only",
			external:  FilterExternal{ExecArgs: "--hash {{.TorrentHash}}"},
			wantBytes: true,
		},
		{
			name:     "unrelated macros need nothing",
			external: FilterExternal{ExecArgs: "--name {{.TorrentName}}", WebhookData: `{"indexer":"{{.Indexer}}"}`},
		},
		{
			name:     "empty",
			external: FilterExternal{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantBytes, tt.external.NeedTorrentDownloaded(), "needs contents")
			assert.Equal(t, tt.wantTmpFil, tt.external.NeedTorrentTmpFile(), "needs tmp file")
		})
	}
}
