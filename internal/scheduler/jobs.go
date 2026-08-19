// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/events"

	"github.com/dustin/go-humanize"
	"github.com/rs/zerolog"
)

type eventBus interface {
	EmitAppUpdate(ctx context.Context, event events.AppUpdateEvent)
}

type CheckUpdatesJob struct {
	name          string
	log           zerolog.Logger
	eventBus      eventBus
	version       string
	updateService updateChecker

	lastCheckVersion string
}

func NewUpdateCheckerJob(log zerolog.Logger, bus eventBus, name, version string, updateService updateChecker) *CheckUpdatesJob {
	return &CheckUpdatesJob{
		log:              log.With().Str("job", name).Logger(),
		eventBus:         bus,
		name:             name,
		version:          version,
		lastCheckVersion: version,
		updateService:    updateService,
	}
}

func (j *CheckUpdatesJob) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	newRelease, err := j.updateService.CheckUpdateAvailable(ctx)
	if err != nil {
		j.log.Error().Err(err).Msg("could not check for new release")
		return
	}

	if newRelease != nil {
		// this is not persisted so this can trigger more than once
		// lets check if we have different versions between runs
		if newRelease.TagName != j.lastCheckVersion {
			j.log.Info().Str("version", newRelease.TagName).Msg("new release available")

			j.eventBus.EmitAppUpdate(ctx, events.AppUpdateEvent{
				Event:          events.Event{Type: events.ApplicationUpdate},
				CurrentVersion: j.version,
				NewVersion:     newRelease.TagName,
				URL:            newRelease.HtmlURL,
			})
		}

		j.lastCheckVersion = newRelease.TagName
	}
}

type TempDirCleanupJob struct {
	Name string
	log  zerolog.Logger
}

func NewTempDirCleanupJob(log zerolog.Logger) *TempDirCleanupJob {
	return &TempDirCleanupJob{
		Name: "temp-dir-cleanup",
		log:  log,
	}
}

func (j *TempDirCleanupJob) Run() {
	var deletedCount uint
	var totalSize uint64

	j.log.Debug().Msg("Starting cleanup of temporary directory.")

	tmpFilePattern := "autobrr-"
	tmpDir := os.TempDir()

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		j.log.Error().Err(err).Str("tmpDir", tmpDir).Msg("failed to read temporary directory")
		return
	}

	// Ask the OS rather than the environment. UID is a shell variable that is not
	// exported to child processes and USER is a name, not a numeric id. Compare
	// against the effective uid since that is what files we create are owned by.
	currentUID := os.Geteuid()

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), tmpFilePattern) {
			continue
		}

		tempFile := filepath.Join(tmpDir, file.Name())

		// Info reports on the directory entry itself. os.Stat would follow a
		// symlink and we would end up judging the target's owner and mtime,
		// which on a shared /tmp is another user's to control.
		fileInfo, err := file.Info()
		if err != nil {
			if os.IsNotExist(err) {
				// removed since the directory was read, nothing to do
				continue
			}

			j.log.Error().Err(err).Str("file", tempFile).Msg("failed to get file info")
			continue
		}

		// only ever touch regular files, a symlink or directory sharing the
		// prefix is not ours to reason about
		if !fileInfo.Mode().IsRegular() {
			continue
		}

		// on a shared box other users keep their torrents in the same /tmp
		if !isOwnedByCurrentUser(currentUID, fileInfo) {
			continue
		}

		if fileInfo.ModTime().Before(time.Now().Add(-24 * time.Hour)) {
			fileSize := uint64(fileInfo.Size())
			if err := os.Remove(tempFile); err != nil {
				j.log.Error().Err(err).Str("file", tempFile).Msg("failed to remove temporary file")
				continue
			}
			j.log.Trace().Str("file", tempFile).Msg("removed file")
			deletedCount++
			totalSize += fileSize
		}
	}

	j.log.Debug().Uint("deleted_count", deletedCount).Str("total_size", humanize.IBytes(totalSize)).Msg("completed temp directory cleanup")
}
