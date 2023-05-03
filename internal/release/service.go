// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package release

import (
	"context"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/rs/zerolog"
)

type Service interface {
	Find(ctx context.Context, query domain.ReleaseQueryParams) (res []*domain.Release, nextCursor int64, count int64, err error)
	FindRecent(ctx context.Context) ([]*domain.Release, error)
	GetIndexerOptions(ctx context.Context) ([]string, error)
	Stats(ctx context.Context) (*domain.ReleaseStats, error)
	Store(ctx context.Context, release *domain.Release) error
	StoreReleaseActionStatus(ctx context.Context, actionStatus *domain.ReleaseActionStatus) error
	Delete(ctx context.Context) error

	Process(release *domain.Release)
	ProcessMultiple(releases []*domain.Release)
}

type actionClientTypeKey struct {
	Type     domain.ActionType
	ClientID int32
}

type service struct {
	log  zerolog.Logger
	repo domain.ReleaseRepo

	actionSvc action.Service
	filterSvc filter.Service
}

func NewService(log logger.Logger, repo domain.ReleaseRepo, actionSvc action.Service, filterSvc filter.Service) Service {
	return &service{
		log:       log.With().Str("module", "release").Logger(),
		repo:      repo,
		actionSvc: actionSvc,
		filterSvc: filterSvc,
	}
}

func (s *service) Find(ctx context.Context, query domain.ReleaseQueryParams) (res []*domain.Release, nextCursor int64, count int64, err error) {
	return s.repo.Find(ctx, query)
}

func (s *service) FindRecent(ctx context.Context) (res []*domain.Release, err error) {
	return s.repo.FindRecent(ctx)
}

func (s *service) GetIndexerOptions(ctx context.Context) ([]string, error) {
	return s.repo.GetIndexerOptions(ctx)
}

func (s *service) Stats(ctx context.Context) (*domain.ReleaseStats, error) {
	return s.repo.Stats(ctx)
}

func (s *service) Store(ctx context.Context, release *domain.Release) error {
	_, err := s.repo.Store(ctx, release)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) StoreReleaseActionStatus(ctx context.Context, status *domain.ReleaseActionStatus) error {
	return s.repo.StoreReleaseActionStatus(ctx, status)
}

func (s *service) Delete(ctx context.Context) error {
	return s.repo.Delete(ctx)
}

func (s *service) Process(release *domain.Release) {
	if release == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			s.log.Error().Msgf("recovering from panic in release process %s error: %v", release.TorrentName, r)
			//err := errors.New("panic in release process: %s", release.TorrentName)
			return
		}
	}()

	defer release.CleanupTemporaryFiles()

	ctx := context.Background()

	// TODO check in config for "Save all releases"
	// TODO cross-seed check
	// TODO dupe checks

	// get filters by priority
	filters, err := s.filterSvc.FindByIndexerIdentifier(ctx, release.Indexer)
	if err != nil {
		s.log.Error().Err(err).Msgf("release.Process: error finding filters for indexer: %s", release.Indexer)
		return
	}

	if len(filters) == 0 {
		s.log.Warn().Msgf("no active filters found for indexer: %s", release.Indexer)
		return
	}

	// keep track of action clients to avoid sending the same thing all over again
	// save both client type and client id to potentially try another client of same type
	triedActionClients := map[actionClientTypeKey]struct{}{}

	// loop over and check filters
	for _, f := range filters {
		l := s.log.With().Str("indexer", release.Indexer).Str("filter", f.Name).Str("release", release.TorrentName).Logger()

		// save filter on release
		release.Filter = &f
		release.FilterName = f.Name
		release.FilterID = f.ID

		// test filter
		match, err := s.filterSvc.CheckFilter(ctx, f, release)
		if err != nil {
			l.Error().Err(err).Msg("release.Process: error checking filter")
			return
		}

		if !match {
			l.Trace().Msgf("release.Process: indexer: %s, filter: %s release: %s, no match. rejections: %s", release.Indexer, release.Filter.Name, release.TorrentName, release.RejectionsString())

			l.Debug().Msgf("release rejected: %s", release.RejectionsString())
			continue
		}

		l.Info().Msgf("Matched '%s' (%s) for %s", release.TorrentName, release.Filter.Name, release.Indexer)

		// save release here to only save those with rejections from actions instead of all releases
		if release.ID == 0 {
			release.FilterStatus = domain.ReleaseStatusFilterApproved
			if err = s.Store(ctx, release); err != nil {
				l.Error().Err(err).Msgf("release.Process: error writing release to database: %+v", release)
				return
			}
		}

		// sleep for the delay period specified in the filter before running actions
		delay := release.Filter.Delay
		if delay > 0 {
			l.Debug().Msgf("Delaying processing of '%s' (%s) for %s by %d seconds as specified in the filter", release.TorrentName, release.Filter.Name, release.Indexer, delay)
			time.Sleep(time.Duration(delay) * time.Second)
		}

		var rejections []string

		// run actions (watchFolder, test, exec, qBittorrent, Deluge, arr etc.)
		for _, a := range release.Filter.Actions {
			act := a

			// only run enabled actions
			if !act.Enabled {
				l.Trace().Msgf("release.Process: indexer: %s, filter: %s release: %s action '%s' not enabled, skip", release.Indexer, release.Filter.Name, release.TorrentName, act.Name)
				continue
			}

			l.Trace().Msgf("release.Process: indexer: %s, filter: %s release: %s , run action: %s", release.Indexer, release.Filter.Name, release.TorrentName, act.Name)

			// keep track of actiom clients to avoid sending the same thing all over again
			_, tried := triedActionClients[actionClientTypeKey{Type: act.Type, ClientID: act.ClientID}]
			if tried {
				l.Trace().Msgf("release.Process: indexer: %s, filter: %s release: %s action client already tried, skip", release.Indexer, release.Filter.Name, release.TorrentName)
				continue
			}

			// run action
			status, err := s.runAction(ctx, act, release)
			if err != nil {
				l.Error().Stack().Err(err).Msgf("release.Process: error running actions for filter: %s", release.Filter.Name)
				//continue
			}

			rejections = status.Rejections

			if err := s.StoreReleaseActionStatus(ctx, status); err != nil {
				s.log.Error().Err(err).Msgf("release.Process: error storing action status for filter: %s", release.Filter.Name)
			}

			if len(rejections) > 0 {
				// if we get action rejection, remember which action client it was from
				triedActionClients[actionClientTypeKey{Type: act.Type, ClientID: act.ClientID}] = struct{}{}

				// log something and fire events
				l.Debug().Str("action", act.Name).Str("action_type", string(act.Type)).Msgf("release rejected: %s", strings.Join(rejections, ", "))
			}

			// if no rejections consider action approved, run next
			continue
		}

		// if we have rejections from arr, continue to next filter
		if len(rejections) > 0 {
			continue
		}

		// all actions run, decide to stop or continue here
		break
	}

	return
}

func (s *service) ProcessMultiple(releases []*domain.Release) {
	s.log.Debug().Msgf("process (%v) new releases from feed", len(releases))

	for _, rls := range releases {
		rls := rls
		if rls == nil {
			continue
		}
		s.Process(rls)
	}
}

func (s *service) runAction(ctx context.Context, action *domain.Action, release *domain.Release) (*domain.ReleaseActionStatus, error) {
	// add action status as pending
	status := domain.NewReleaseActionStatus(action, release)

	if err := s.StoreReleaseActionStatus(ctx, status); err != nil {
		s.log.Error().Err(err).Msgf("release.runAction: error storing action for filter: %s", release.Filter.Name)
	}

	rejections, err := s.actionSvc.RunAction(ctx, action, release)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("release.runAction: error running actions for filter: %s", release.Filter.Name)

		status.Status = domain.ReleasePushStatusErr
		status.Rejections = []string{err.Error()}

		return status, err
	}

	if rejections != nil {
		status.Status = domain.ReleasePushStatusRejected
		status.Rejections = rejections

		return status, nil
	}

	status.Status = domain.ReleasePushStatusApproved

	return status, nil
}
