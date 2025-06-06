// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package release

import (
	"context"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

type Service interface {
	Find(ctx context.Context, query domain.ReleaseQueryParams) (*domain.FindReleasesResponse, error)
	Get(ctx context.Context, req *domain.GetReleaseRequest) (*domain.Release, error)
	GetActionStatus(ctx context.Context, req *domain.GetReleaseActionStatusRequest) (*domain.ReleaseActionStatus, error)
	GetIndexerOptions(ctx context.Context) ([]string, error)
	Stats(ctx context.Context) (*domain.ReleaseStats, error)
	Store(ctx context.Context, release *domain.Release) error
	Update(ctx context.Context, release *domain.Release) error
	StoreReleaseActionStatus(ctx context.Context, actionStatus *domain.ReleaseActionStatus) error
	Delete(ctx context.Context, req *domain.DeleteReleaseRequest) error
	Process(release *domain.Release)
	ProcessMultiple(releases []*domain.Release)
	ProcessMultipleFromIndexer(releases []*domain.Release, indexer domain.IndexerMinimal) error
	ProcessManual(ctx context.Context, req *domain.ReleaseProcessReq) error
	Retry(ctx context.Context, req *domain.ReleaseActionRetryReq) error

	StoreReleaseProfileDuplicate(ctx context.Context, profile *domain.DuplicateReleaseProfile) error
	FindDuplicateReleaseProfiles(ctx context.Context) ([]*domain.DuplicateReleaseProfile, error)
	DeleteReleaseProfileDuplicate(ctx context.Context, id int64) error
}

type actionClientTypeKey struct {
	Type     domain.ActionType
	ClientID int32
}

type service struct {
	log  zerolog.Logger
	repo domain.ReleaseRepo

	actionSvc  action.Service
	filterSvc  filter.Service
	indexerSvc indexer.Service
}

func NewService(log logger.Logger, repo domain.ReleaseRepo, actionSvc action.Service, filterSvc filter.Service, indexerSvc indexer.Service) Service {
	return &service{
		log:        log.With().Str("module", "release").Logger(),
		repo:       repo,
		actionSvc:  actionSvc,
		filterSvc:  filterSvc,
		indexerSvc: indexerSvc,
	}
}

func (s *service) Find(ctx context.Context, query domain.ReleaseQueryParams) (*domain.FindReleasesResponse, error) {
	return s.repo.Find(ctx, query)
}

func (s *service) Get(ctx context.Context, req *domain.GetReleaseRequest) (*domain.Release, error) {
	return s.repo.Get(ctx, req)
}

func (s *service) GetActionStatus(ctx context.Context, req *domain.GetReleaseActionStatusRequest) (*domain.ReleaseActionStatus, error) {
	return s.repo.GetActionStatus(ctx, req)
}

func (s *service) GetIndexerOptions(ctx context.Context) ([]string, error) {
	return s.repo.GetIndexerOptions(ctx)
}

func (s *service) Stats(ctx context.Context) (*domain.ReleaseStats, error) {
	return s.repo.Stats(ctx)
}

func (s *service) Store(ctx context.Context, release *domain.Release) error {
	return s.repo.Store(ctx, release)
}

func (s *service) Update(ctx context.Context, release *domain.Release) error {
	return s.repo.Update(ctx, release)
}

func (s *service) StoreReleaseActionStatus(ctx context.Context, status *domain.ReleaseActionStatus) error {
	return s.repo.StoreReleaseActionStatus(ctx, status)
}

func (s *service) Delete(ctx context.Context, req *domain.DeleteReleaseRequest) error {
	return s.repo.Delete(ctx, req)
}

func (s *service) FindDuplicateReleaseProfiles(ctx context.Context) ([]*domain.DuplicateReleaseProfile, error) {
	return s.repo.FindDuplicateReleaseProfiles(ctx)
}

func (s *service) StoreReleaseProfileDuplicate(ctx context.Context, profile *domain.DuplicateReleaseProfile) error {
	return s.repo.StoreDuplicateProfile(ctx, profile)
}

func (s *service) DeleteReleaseProfileDuplicate(ctx context.Context, id int64) error {
	return s.repo.DeleteReleaseProfileDuplicate(ctx, id)
}

func (s *service) ProcessManual(ctx context.Context, req *domain.ReleaseProcessReq) error {
	// get indexer definition with data
	def, err := s.indexerSvc.GetMappedDefinitionByName(req.IndexerIdentifier)
	if err != nil {
		return err
	}

	rls := domain.NewRelease(domain.IndexerMinimal{ID: def.ID, Name: def.Name, Identifier: def.Identifier, IdentifierExternal: def.IdentifierExternal})

	switch req.IndexerImplementation {
	case string(domain.IndexerImplementationIRC):

		// from announce/announce.go
		tmpVars := map[string]string{}
		parseFailed := false

		for idx, parseLine := range def.IRC.Parse.Lines {
			match, err := indexer.ParseLine(&s.log, parseLine.Pattern, parseLine.Vars, tmpVars, req.AnnounceLines[idx], parseLine.Ignore)
			if err != nil {
				parseFailed = true
				break
			}

			if !match {
				parseFailed = true
				break
			}
		}

		if parseFailed {
			return errors.New("parse failed")
		}

		rls.Protocol = domain.ReleaseProtocol(def.Protocol)

		// on lines matched
		err = def.IRC.Parse.Parse(def, tmpVars, rls)
		if err != nil {
			return err
		}

	default:
		return errors.New("implementation %q is not supported", req.IndexerImplementation)

	}

	// process
	go s.Process(rls)

	return nil
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

	ctx := context.Background()

	// TODO check in config for "Save all releases"
	// TODO cross-seed check
	// TODO dupe checks

	// get filters by priority
	filters, err := s.filterSvc.FindByIndexerIdentifier(ctx, release.Indexer.Identifier)
	if err != nil {
		s.log.Error().Err(err).Msgf("release.Process: error finding filters for indexer: %s", release.Indexer.Name)
		return
	}

	if len(filters) == 0 {
		s.log.Warn().Msgf("no active filters found for indexer: %s", release.Indexer.Name)
		return
	}

	if err := s.processRelease(ctx, release, filters); err != nil {
		s.log.Error().Err(err).Msgf("release.Process: error processing filters for indexer: %s", release.Indexer.Name)
		return
	}

	return
}

func (s *service) processRelease(ctx context.Context, release *domain.Release, filters []*domain.Filter) error {
	defer release.CleanupTemporaryFiles()

	if err := s.processFilters(ctx, filters, release); err != nil {
		s.log.Error().Err(err).Msgf("release.Process: error processing filters for indexer: %s", release.Indexer.Name)
		return err
	}

	return nil
}

func (s *service) processFilters(ctx context.Context, filters []*domain.Filter, release *domain.Release) error {
	// keep track of action clients to avoid sending the same thing all over again
	// save both client type and client id to potentially try another client of same type
	triedActionClients := map[actionClientTypeKey]struct{}{}

	// loop over and check filters
	for _, f := range filters {
		l := s.log.With().Str("indexer", release.Indexer.Identifier).Str("filter", f.Name).Str("release", release.TorrentName).Logger()

		// save filter on release
		release.Filter = f
		release.FilterName = f.Name
		release.FilterID = f.ID

		// reset IsDuplicate
		release.IsDuplicate = false
		release.SkipDuplicateProfileID = 0
		release.SkipDuplicateProfileName = ""

		// test filter
		match, err := s.filterSvc.CheckFilter(ctx, f, release)
		if err != nil {
			l.Error().Err(err).Msg("release.Process: error checking filter")
			return err
		}

		if !match || f.RejectReasons.Len() > 0 {
			l.Trace().Msgf("release.Process: indexer: %s, filter: %s release: %s, no match. rejections: %s", release.Indexer.Name, release.FilterName, release.TorrentName, f.RejectReasons.String())

			l.Debug().Msgf("filter %s rejected release: %s with reasons: %s", f.Name, release.TorrentName, f.RejectReasons.StringTruncated())
			continue
		}

		l.Info().Msgf("Matched '%s' (%s) for %s", release.TorrentName, release.FilterName, release.Indexer.Name)

		// found matching filter, lets find the filter actions and attach
		active := true
		actions, err := s.actionSvc.FindByFilterID(ctx, f.ID, &active, false)
		if err != nil {
			s.log.Error().Err(err).Msgf("release.Process: error finding actions for filter: %s", f.Name)
			return err
		}

		// if no actions, continue to next filter
		if len(actions) == 0 {
			s.log.Warn().Msgf("release.Process: no active actions found for filter '%s', trying next one..", f.Name)
			continue
		}

		// save release here to only save those with rejections from actions instead of all releases
		if release.ID == 0 {
			release.FilterStatus = domain.ReleaseStatusFilterApproved

			if err = s.Store(ctx, release); err != nil {
				l.Error().Err(err).Msgf("release.Process: error writing release to database: %+v", release)
				return err
			}
		}

		var rejections []string

		// run actions (watchFolder, test, exec, qBittorrent, Deluge, arr etc.)
		for idx, act := range actions {
			// only run enabled actions
			if !act.Enabled {
				l.Trace().Msgf("release.Process: indexer: %s, filter: %s release: %s action '%s' not enabled, skip", release.Indexer.Name, release.FilterName, release.TorrentName, act.Name)
				continue
			}

			// add action status as pending
			actionStatus := domain.NewReleaseActionStatus(act, release)

			if err := s.StoreReleaseActionStatus(ctx, actionStatus); err != nil {
				s.log.Error().Err(err).Msgf("release.runAction: error storing action for filter: %s", release.FilterName)
			}

			if idx == 0 {
				// sleep for the delay period specified in the filter before running actions
				delay := release.Filter.Delay
				if delay > 0 {
					l.Debug().Msgf("release.Process: delaying processing of '%s' (%s) for %s by %d seconds as specified in the filter", release.TorrentName, release.FilterName, release.Indexer.Name, delay)
					time.Sleep(time.Duration(delay) * time.Second)
				}
			}

			l.Trace().Msgf("release.Process: indexer: %s, filter: %s release: %s , run action: %s", release.Indexer.Name, release.FilterName, release.TorrentName, act.Name)

			// keep track of action clients to avoid sending the same thing all over again
			_, tried := triedActionClients[actionClientTypeKey{Type: act.Type, ClientID: act.ClientID}]
			if tried {
				l.Debug().Msgf("release.Process: indexer: %s, filter: %s release: %s action client already tried, skip", release.Indexer.Name, release.FilterName, release.TorrentName)
				continue
			}

			// run action
			status, err := s.runAction(ctx, act, release, actionStatus)
			if err != nil {
				l.Error().Err(err).Msgf("release.Process: error running actions for filter: %s", release.FilterName)
				//continue
			}

			rejections = status.Rejections

			if err := s.StoreReleaseActionStatus(ctx, status); err != nil {
				s.log.Error().Err(err).Msgf("release.Process: error storing action status for filter: %s", release.FilterName)
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

		if err = s.Update(ctx, release); err != nil {
			l.Error().Err(err).Msgf("release.Process: error updating release: %v", release.TorrentName)
		}

		// if we have rejections from arr, continue to next filter
		if len(rejections) > 0 {
			continue
		}

		// all actions run, decide to stop or continue here
		break
	}

	return nil
}

func (s *service) ProcessMultiple(releases []*domain.Release) {
	s.log.Debug().Msgf("process (%d) new releases from feed", len(releases))

	for _, rls := range releases {
		if rls == nil {
			continue
		}
		s.Process(rls)
	}
}

func (s *service) ProcessMultipleFromIndexer(releases []*domain.Release, indexer domain.IndexerMinimal) error {
	s.log.Debug().Msgf("process (%d) new releases from feed %s", len(releases), indexer.Name)

	defer func() {
		if r := recover(); r != nil {
			s.log.Error().Msgf("recovering from panic in release process %s error: %v", "", r)
			//err := errors.New("panic in release process: %s", release.TorrentName)
			return
		}
	}()

	ctx := context.Background()

	// get filters by priority
	filters, err := s.filterSvc.FindByIndexerIdentifier(ctx, indexer.Identifier)
	if err != nil {
		s.log.Error().Err(err).Msgf("release.Process: error finding filters for indexer: %s", indexer.Name)
		return err
	}

	if len(filters) == 0 {
		s.log.Warn().Msgf("no active filters found for indexer: %s skipping rest..", indexer.Name)
		return domain.ErrNoActiveFiltersFoundForIndexer
	}

	for _, release := range releases {
		if release == nil {
			continue
		}

		if err := s.processRelease(ctx, release, filters); err != nil {
			s.log.Error().Err(err).Msgf("release.ProcessMultipleFromIndexer: error processing filters for indexer: %s", indexer.Name)
			return nil
		}
	}

	return nil
}

func (s *service) runAction(ctx context.Context, action *domain.Action, release *domain.Release, status *domain.ReleaseActionStatus) (*domain.ReleaseActionStatus, error) {
	// add action status as pending
	//status := domain.NewReleaseActionStatus(action, release)
	//
	//if err := s.StoreReleaseActionStatus(ctx, status); err != nil {
	//	s.log.Error().Err(err).Msgf("release.runAction: error storing action for filter: %s", release.FilterName)
	//}

	rejections, err := s.actionSvc.RunAction(ctx, action, release)
	if err != nil {
		s.log.Error().Err(err).Msgf("release.runAction: error running actions for filter: %s", release.FilterName)

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

func (s *service) retryAction(ctx context.Context, action *domain.Action, release *domain.Release) error {
	// add action status as pending
	status := domain.NewReleaseActionStatus(action, release)

	if err := s.StoreReleaseActionStatus(ctx, status); err != nil {
		s.log.Error().Err(err).Msgf("release.runAction: error storing action for filter: %s", release.FilterName)
	}

	actionStatus, err := s.runAction(ctx, action, release, status)
	if err != nil {
		s.log.Error().Err(err).Msgf("release.retryAction: error running actions for filter: %s", release.FilterName)

		if err := s.StoreReleaseActionStatus(ctx, actionStatus); err != nil {
			s.log.Error().Err(err).Msgf("release.retryAction: error storing filterAction status for filter: %s", release.FilterName)
			return err
		}

		return err
	}

	if err := s.StoreReleaseActionStatus(ctx, actionStatus); err != nil {
		s.log.Error().Err(err).Msgf("release.retryAction: error storing filterAction status for filter: %s", release.FilterName)
		return err
	}

	return nil
}

func (s *service) Retry(ctx context.Context, req *domain.ReleaseActionRetryReq) error {
	// get release
	release, err := s.Get(ctx, &domain.GetReleaseRequest{Id: req.ReleaseId})
	if err != nil {
		return errors.Wrap(err, "retry error: could not find release by id: %d", req.ReleaseId)
	}

	indexerInfo, err := s.indexerSvc.GetBy(ctx, domain.GetIndexerRequest{Identifier: release.Indexer.Identifier})
	if err != nil {
		return errors.Wrap(err, "retry error: could not get indexer by identifier: %s", release.Indexer.Identifier)
	}

	release.Indexer = domain.IndexerMinimal{
		ID:                 int(indexerInfo.ID),
		Name:               indexerInfo.Name,
		Identifier:         indexerInfo.Identifier,
		IdentifierExternal: indexerInfo.IdentifierExternal,
	}

	// get release filter action status
	status, err := s.GetActionStatus(ctx, &domain.GetReleaseActionStatusRequest{Id: req.ActionStatusId})
	if err != nil {
		return errors.Wrap(err, "retry error: could not get release action")
	}

	// get filter action with action id from status
	filterAction, err := s.actionSvc.Get(ctx, &domain.GetActionRequest{Id: int(status.ActionID)})
	if err != nil {
		return errors.Wrap(err, "retry error: could not get filter action for release")
	}

	// run filterAction
	if err := s.retryAction(ctx, filterAction, release); err != nil {
		s.log.Error().Err(err).Msgf("release.Retry: error re-running action: %s", filterAction.Name)
		return err
	}

	s.log.Info().Msgf("successfully replayed action %s for release %s", filterAction.Name, release.TorrentName)

	return nil
}
