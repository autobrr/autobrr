// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package release

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
)

type CleanupJob struct {
	log            zerolog.Logger
	releaseRepo    domain.ReleaseRepo
	cleanupJobRepo domain.ReleaseCleanupJobRepo
	job            *domain.ReleaseCleanupJob
}

func NewCleanupJob(log zerolog.Logger, releaseRepo domain.ReleaseRepo, cleanupJobRepo domain.ReleaseCleanupJobRepo, job *domain.ReleaseCleanupJob) *CleanupJob {
	return &CleanupJob{
		log:            log,
		releaseRepo:    releaseRepo,
		cleanupJobRepo: cleanupJobRepo,
		job:            job,
	}
}

func (j *CleanupJob) Run() {
	ctx := context.Background()
	j.log.Debug().Str("job", j.job.Name).Msg("running release cleanup job")

	// Track execution time and status
	startTime := time.Now()
	j.job.LastRun = startTime

	// Parse comma-separated indexers
	var indexers []string
	if j.job.Indexers != "" {
		for idx := range strings.SplitSeq(j.job.Indexers, ",") {
			trimmed := strings.TrimSpace(idx)
			if trimmed != "" {
				indexers = append(indexers, trimmed)
			}
		}
	}

	// Parse and validate comma-separated statuses
	var statuses []string
	if j.job.Statuses != "" {
		for s := range strings.SplitSeq(j.job.Statuses, ",") {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				if domain.ValidDeletableReleasePushStatus(trimmed) {
					statuses = append(statuses, trimmed)
				} else {
					j.log.Warn().Str("status", trimmed).Msg("invalid release status ignored")
				}
			}
		}
	}

	// Build delete request
	req := &domain.DeleteReleaseRequest{
		OlderThan:       j.job.OlderThan,
		Indexers:        indexers,
		ReleaseStatuses: statuses,
	}

	// Perform deletions
	if err := j.releaseRepo.Delete(ctx, req); err != nil {
		j.log.Error().Err(err).Msg("error deleting releases")

		// Update job with error status
		j.job.LastRunStatus = domain.ReleaseCleanupStatusError
		j.job.LastRunData = err.Error()
		if err := j.cleanupJobRepo.UpdateLastRun(ctx, j.job); err != nil {
			j.log.Error().Err(err).Msg("error updating cleanup job status")
		}
		return
	}

	// Build success data
	successData := map[string]any{
		"older_than_hours": req.OlderThan,
		"indexers":         indexers,
		"statuses":         statuses,
		"duration_ms":      time.Since(startTime).Milliseconds(),
	}

	dataJSON, err := json.Marshal(successData)
	if err != nil {
		j.log.Error().Err(err).Msg("error marshaling success data")
		dataJSON = []byte("{}")
	}

	// Update job with success status
	j.job.LastRunStatus = domain.ReleaseCleanupStatusSuccess
	j.job.LastRunData = string(dataJSON)
	if err := j.cleanupJobRepo.UpdateLastRun(ctx, j.job); err != nil {
		j.log.Error().Err(err).Msg("error updating cleanup job status")
	}

	j.log.Info().
		Str("job", j.job.Name).
		Int("older_than_hours", req.OlderThan).
		Strs("indexers", indexers).
		Strs("statuses", statuses).
		Msg("release cleanup completed successfully")
}
