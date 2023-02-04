package update

import (
	"context"
	"sync"

	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/version"

	"github.com/rs/zerolog"
)

type Service struct {
	log     zerolog.Logger
	version string

	m             sync.RWMutex
	latestRelease *version.Release
}

func NewUpdate(log logger.Logger, currentVersion string) *Service {
	return &Service{
		log:     log.With().Str("module", "update").Logger(),
		version: currentVersion,
	}
}

func (s *Service) AvailableRelease(ctx context.Context) *version.Release {
	s.m.RLock()
	defer s.m.RUnlock()
	return s.latestRelease
}

func (s *Service) CheckUpdates(ctx context.Context) {
	v := version.Checker{
		Owner:          "autobrr",
		Repo:           "autobrr",
		CurrentVersion: s.version,
	}

	newAvailable, newVersion, err := v.CheckNewVersion(ctx, s.version)
	if err != nil {
		s.log.Error().Err(err).Msg("could not check for new release")
		return
	}

	if newAvailable {
		s.log.Info().Msgf("autobrr outdated, found new release: %s", newVersion.TagName)

		s.m.Lock()
		defer s.m.Unlock()

		s.latestRelease = newVersion
	}

	return
}
