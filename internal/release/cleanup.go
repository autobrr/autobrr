// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package release

import (
	"context"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
)

type CleanupJob struct {
	log    zerolog.Logger
	repo   domain.ReleaseRepo
	config *domain.Config
}

func NewCleanupJob(log zerolog.Logger, repo domain.ReleaseRepo, config *domain.Config) *CleanupJob {
	return &CleanupJob{
		log:    log,
		repo:   repo,
		config: config,
	}
}

func (j *CleanupJob) Run() {
	j.log.Debug().Msg("running release-cleanup job")

	if !j.config.ReleaseCleanupEnabled {
		j.log.Debug().Msg("release cleanup is disabled")
		return
	}

	// Parse comma-separated indexers
	var indexers []string
	if j.config.ReleaseCleanupIndexers != "" {
		rawIndexers := strings.Split(j.config.ReleaseCleanupIndexers, ",")
		for _, idx := range rawIndexers {
			trimmed := strings.TrimSpace(idx)
			if trimmed != "" {
				indexers = append(indexers, trimmed)
			}
		}
	}

	// Parse and validate comma-separated statuses
	var statuses []string
	if j.config.ReleaseCleanupStatuses != "" {
		rawStatuses := strings.Split(j.config.ReleaseCleanupStatuses, ",")
		for _, s := range rawStatuses {
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
		OlderThan:       j.config.ReleaseCleanupOlderThan,
		Indexers:        indexers,
		ReleaseStatuses: statuses,
	}

	// Perform deletions
	if err := j.repo.Delete(context.Background(), req); err != nil {
		j.log.Error().Err(err).Msg("error deleting releases")
		return
	}

	j.log.Info().
		Int("older_than_hours", req.OlderThan).
		Strs("indexers", indexers).
		Strs("statuses", statuses).
		Msg("release cleanup completed successfully")
}
