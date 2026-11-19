// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package update

import (
	"context"
	"sync"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/internal/meta"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/version"

	"github.com/rs/zerolog"
)

type eventBus interface {
	EmitAppUpdate(ctx context.Context, event events.AppUpdateEvent)
}

type releaseChecker interface {
	CheckNewVersion(ctx context.Context, version string) (bool, *version.Release, error)
}

type Service struct {
	log      zerolog.Logger
	eventBus eventBus
	config   *domain.Config

	m              sync.RWMutex
	releaseChecker releaseChecker
	latestRelease  *version.Release
}

func NewUpdate(log zerolog.Logger, eventBus eventBus, config *domain.Config) *Service {
	return &Service{
		log:            log.With().Str("module", "update").Logger(),
		eventBus:       eventBus,
		config:         config,
		releaseChecker: version.NewChecker("autobrr", "autobrr", meta.GetVersion()),
	}
}

func (s *Service) GetLatestRelease(_ context.Context) *version.Release {
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

// CheckUpdateAvailable returns the latest release when it is newer than the
// running version. The first check to find a given release emits an
// AppUpdateEvent, whichever caller made it.
func (s *Service) CheckUpdateAvailable(ctx context.Context) (*version.Release, error) {
	s.log.Trace().Msg("checking for updates...")

	newAvailable, newVersion, err := s.releaseChecker.CheckNewVersion(ctx, meta.GetVersion())
	if err != nil {
		return nil, errors.Wrap(err, "could not check for new release")
	}

	if !newAvailable {
		return nil, nil
	}

	s.m.Lock()
	found := s.latestRelease == nil || s.latestRelease.TagName != newVersion.TagName
	s.latestRelease = newVersion
	s.m.Unlock()

	// emitted outside the lock, subscribers read the release back through GetLatestRelease
	if found {
		s.log.Info().Str("version", newVersion.TagName).Msg("found newer release")

		s.eventBus.EmitAppUpdate(ctx, events.AppUpdateEvent{
			Type:           events.ApplicationUpdate,
			CurrentVersion: meta.GetVersion(),
			NewVersion:     newVersion.TagName,
			URL:            newVersion.HtmlURL,
		})
	}

	return newVersion, nil
}
