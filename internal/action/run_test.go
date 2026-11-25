// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// fakeDownloadService mirrors DownloadService.DownloadRelease: it only reaches
// the indexer when the release does not already carry the torrent, so a test
// can count actual fetches rather than calls.
type fakeDownloadService struct {
	fetches int
}

func (f *fakeDownloadService) DownloadRelease(_ context.Context, rls *domain.Release) error {
	if len(rls.TorrentDataRawBytes) != 0 {
		return nil
	}

	f.fetches++
	rls.TorrentDataRawBytes = []byte("d8:announce0:4:infod4:name4:test6:lengthi1eee")
	rls.TorrentHash = "abc123"

	return nil
}

func (f *fakeDownloadService) ResolveMagnetURI(_ context.Context, _ *domain.Release) error {
	return nil
}

func newTestService(dl rlsDownloadService) *Service {
	return &Service{log: zerolog.Nop(), rlsDownloadSvc: dl}
}

// Torrent-client actions without torrent-content macros must defer downloading
// until after the client executor checks rules (such as max active downloads),
// so rejected releases do not burn indexer download quota.
func TestCheckActionPreconditions_DefersDownloadForClientActionsWithoutTorrentMacros(t *testing.T) {
	t.Parallel()

	dl := &fakeDownloadService{}
	svc := newTestService(dl)

	release := &domain.Release{TorrentName: "Test.Release-GRP", Protocol: domain.ReleaseProtocolTorrent}

	actions := []*domain.Action{
		{Name: "qbit", Type: domain.ActionTypeQbittorrent},
		{Name: "deluge", Type: domain.ActionTypeDelugeV2},
		{Name: "transmission", Type: domain.ActionTypeTransmission},
		{Name: "porla", Type: domain.ActionTypePorla},
		{Name: "aria2", Type: domain.ActionTypeAria2},
		{Name: "rtorrent", Type: domain.ActionTypeRTorrent},
	}

	for _, action := range actions {
		err := svc.CheckActionPreconditions(t.Context(), action, release)
		assert.NoError(t, err)
	}

	assert.Equal(t, 0, dl.fetches, "client actions without torrent macros must not pre-download before client rule checks")
	assert.Empty(t, release.TorrentDataRawBytes)
}

// Watch-folder actions and actions whose macros reference torrent bytes/hash
// still fetch the torrent once onto the shared release before ParseMacros runs.
func TestCheckActionPreconditions_DownloadsOncePerRelease(t *testing.T) {
	t.Parallel()

	dl := &fakeDownloadService{}
	svc := newTestService(dl)

	release := &domain.Release{TorrentName: "Test.Release-GRP", Protocol: domain.ReleaseProtocolTorrent}

	actions := []*domain.Action{
		{Name: "qbit-hash-macro", Type: domain.ActionTypeQbittorrent, SavePath: "/data/{{.TorrentHash}}"},
		{Name: "watch", Type: domain.ActionTypeWatchFolder},
	}

	for _, action := range actions {
		err := svc.CheckActionPreconditions(t.Context(), action, release)
		assert.NoError(t, err)
	}

	assert.Equal(t, 1, dl.fetches, "torrent should be fetched from the indexer once for all actions")
	assert.NotEmpty(t, release.TorrentDataRawBytes, "torrent should land on the shared release")
}

// Magnet releases carry no torrent to fetch, the client is handed the URI.
func TestCheckActionPreconditions_SkipsDownloadForMagnet(t *testing.T) {
	t.Parallel()

	dl := &fakeDownloadService{}
	svc := newTestService(dl)

	release := &domain.Release{
		TorrentName: "Test.Release-GRP",
		Protocol:    domain.ReleaseProtocolTorrent,
		MagnetURI:   "magnet:?xt=urn:btih:abc123",
	}

	err := svc.CheckActionPreconditions(t.Context(), &domain.Action{Type: domain.ActionTypeQbittorrent}, release)

	assert.NoError(t, err)
	assert.Equal(t, 0, dl.fetches, "magnet releases must not trigger a torrent download")
}

// Actions that never touch the torrent should not pull it down.
func TestCheckActionPreconditions_SkipsDownloadForArrActions(t *testing.T) {
	t.Parallel()

	dl := &fakeDownloadService{}
	svc := newTestService(dl)

	release := &domain.Release{TorrentName: "Test.Release-GRP", Protocol: domain.ReleaseProtocolTorrent}

	err := svc.CheckActionPreconditions(t.Context(), &domain.Action{Type: domain.ActionTypeRadarr}, release)

	assert.NoError(t, err)
	assert.Equal(t, 0, dl.fetches)
}

// A path macro needs a real file, and it has to exist before ParseMacros runs.
func TestCheckActionPreconditions_WritesTmpFileForPathMacro(t *testing.T) {
	t.Parallel()

	dl := &fakeDownloadService{}
	svc := newTestService(dl)

	release := &domain.Release{TorrentName: "Test.Release-GRP", Protocol: domain.ReleaseProtocolTorrent}
	action := &domain.Action{Type: domain.ActionTypeExec, ExecArgs: "--torrent {{.TorrentPathName}}"}

	err := svc.CheckActionPreconditions(t.Context(), action, release)
	assert.NoError(t, err)

	assert.Equal(t, 1, dl.fetches)
	assert.NotEmpty(t, release.TorrentTmpFile, "path macro requires a tmp file on disk")

	t.Cleanup(func() {
		_ = release.CleanupTemporaryFiles()
	})
}

// An exec action without torrent macros has nothing to download.
func TestCheckActionPreconditions_SkipsDownloadForPlainExec(t *testing.T) {
	t.Parallel()

	dl := &fakeDownloadService{}
	svc := newTestService(dl)

	release := &domain.Release{TorrentName: "Test.Release-GRP", Protocol: domain.ReleaseProtocolTorrent}
	action := &domain.Action{Type: domain.ActionTypeExec, ExecArgs: "--name {{.TorrentName}}"}

	err := svc.CheckActionPreconditions(t.Context(), action, release)

	assert.NoError(t, err)
	assert.Equal(t, 0, dl.fetches)
	assert.Empty(t, release.TorrentTmpFile)
}
