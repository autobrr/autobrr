// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package update

import (
	"context"
	"sync"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/version"

	"github.com/rs/zerolog"
)

type Service struct {
	log    zerolog.Logger
	config *domain.Config

	m              sync.RWMutex
	releaseChecker *version.Checker
	latestRelease  *version.Release
}

func NewUpdate(log logger.Logger, config *domain.Config) *Service {
	return &Service{
		log:            log.With().Str("module", "update").Logger(),
		config:         config,
		releaseChecker: version.NewChecker("autobrr", "autobrr", config.Version),
	}
}

func (s *Service) GetLatestRelease(ctx context.Context) *version.Release {
	s.m.RLock()
	defer s.m.RUnlock()
	return s.latestRelease
}

func (s *Service) CheckUpdates(ctx context.Context) {
	if _, err := s.CheckUpdateAvailable(ctx); err != nil {
		s.log.Error().Err(err).Msg("error checking new release")
		return
	}

	return
}

func (s *Service) CheckUpdateAvailable(ctx context.Context) (*version.Release, error) {
	s.log.Trace().Msg("checking for updates...")

	newAvailable, newVersion, err := s.releaseChecker.CheckNewVersion(ctx, s.config.Version)
	if err != nil {
		s.log.Error().Err(err).Msg("could not check for new release")
		return nil, nil
	}

	if newAvailable {
		s.log.Info().Msgf("autobrr outdated, found newer release: %s", newVersion.TagName)

		s.m.Lock()
		defer s.m.Unlock()

		if s.latestRelease != nil && s.latestRelease.TagName == newVersion.TagName {
			return nil, nil
		}

		s.latestRelease = newVersion

		return newVersion, nil
	}

	return nil, nil
}
