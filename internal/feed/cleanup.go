// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
)

type CleanupJob struct {
	log       zerolog.Logger
	cacheRepo domain.FeedCacheRepo

	CronSchedule time.Duration
}

func NewCleanupJob(log zerolog.Logger, cacheRepo domain.FeedCacheRepo) *CleanupJob {
	return &CleanupJob{
		log:       log,
		cacheRepo: cacheRepo,
	}
}

func (j *CleanupJob) Run() {
	if err := j.cacheRepo.DeleteStale(context.Background()); err != nil {
		j.log.Error().Err(err).Msg("error when running feed cache cleanup job")
	}

	j.log.Info().Msg("successfully ran feed-cache-cleanup job")
}
