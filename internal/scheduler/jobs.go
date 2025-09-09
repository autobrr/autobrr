// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package scheduler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/update"

	"github.com/dustin/go-humanize"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type CheckUpdatesJob struct {
	Name          string
	Log           zerolog.Logger
	Version       string
	NotifSvc      notification.Service
	updateService *update.Service

	lastCheckVersion string
}

func (j *CheckUpdatesJob) Run() {
	newRelease, err := j.updateService.CheckUpdateAvailable(context.TODO())
	if err != nil {
		j.Log.Error().Err(err).Msg("could not check for new release")
		return
	}

	if newRelease != nil {
		// this is not persisted so this can trigger more than once
		// lets check if we have different versions between runs
		if newRelease.TagName != j.lastCheckVersion {
			j.Log.Info().Msgf("a new release has been found: %v Consider updating.", newRelease.TagName)

			j.NotifSvc.Send(domain.NotificationEventAppUpdateAvailable, domain.NotificationPayload{
				Subject:   "New update available!",
				Message:   newRelease.TagName,
				Event:     domain.NotificationEventAppUpdateAvailable,
				Timestamp: time.Now(),
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

	currentUID := os.Getenv("UID")
	if currentUID == "" {
		// Fallback for systems where UID isn't set
		currentUID = os.Getenv("USER")
		if currentUID == "" {
			log.Debug().Msg("could not determine current user, skipping ownership check")
			// Continue without ownership filtering or implement alternative logic
		}
	}

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), tmpFilePattern) {
			continue
		}

		tempFile := filepath.Join(tmpDir, file.Name())

		fileInfo, err := os.Stat(tempFile)
		if err != nil {
			j.log.Error().Err(err).Str("file", tempFile).Msg("failed to get file info")
			continue
		}

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

	j.log.Debug().Msgf("Completed cleanup of temporary directory. Deleted %d files with a total size of %s.", deletedCount, humanize.IBytes(totalSize))
}
