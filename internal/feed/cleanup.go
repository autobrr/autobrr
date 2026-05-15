// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

type CleanupJob struct {
	log       zerolog.Logger
	cacheRepo feedCacheRepoCleaner

	CronSchedule time.Duration
}

type feedCacheRepoCleaner interface {
	DeleteStale(ctx context.Context) error
	DeleteOrphaned(ctx context.Context) error
}

func NewCleanupJob(log zerolog.Logger, cacheRepo feedCacheRepoCleaner) *CleanupJob {
	return &CleanupJob{
		log:       log,
		cacheRepo: cacheRepo,
	}
}

func (j *CleanupJob) Run() {
	j.log.Info().Msg("running feed-cache-cleanup job..")

	if err := j.cacheRepo.DeleteStale(context.Background()); err != nil {
		j.log.Error().Err(err).Msg("error when running feed cache cleanup job")
		return
	}

	if err := j.cacheRepo.DeleteOrphaned(context.Background()); err != nil {
		j.log.Error().Err(err).Msg("error when running feed cache cleanup job")
		return
	}

	j.log.Info().Msg("successfully ran feed-cache-cleanup job")
}
