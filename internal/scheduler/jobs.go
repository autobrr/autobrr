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

	"github.com/rs/zerolog"
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
	Log  zerolog.Logger
}

func (j *TempDirCleanupJob) Run() {
	j.Log.Debug().Msg("Starting cleanup of temporary directory.")

	tmpFilePattern := "autobrr-"
	tmpDir := os.TempDir()

	files, err := os.ReadDir(tmpDir)
	if err == nil {
		for _, file := range files {
			if strings.HasPrefix(file.Name(), tmpFilePattern) {
				os.Remove(filepath.Join(tmpDir, file.Name()))
			}
		}
	}
}
