// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package release

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/asaskevich/EventBus"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// releaseCleanupJobRepo interface for managing cleanup jobs
type releaseCleanupJobRepo interface {
	ListCleanupJobs(ctx context.Context) ([]*domain.ReleaseCleanupJob, error)
	FindCleanupJobByID(ctx context.Context, id int) (*domain.ReleaseCleanupJob, error)
	StoreCleanupJob(ctx context.Context, job *domain.ReleaseCleanupJob) error
	UpdateCleanupJob(ctx context.Context, job *domain.ReleaseCleanupJob) error
	UpdateCleanupJobLastRun(ctx context.Context, job *domain.ReleaseCleanupJob) error
	CleanupJobToggleEnabled(ctx context.Context, id int, enabled bool) error
	DeleteCleanupJob(ctx context.Context, id int) error
}

type releaseRepo interface {
	Store(ctx context.Context, release *domain.Release) error
	Update(ctx context.Context, r *domain.Release) error
	Find(ctx context.Context, params domain.ReleaseQueryParams) (*domain.FindReleasesResponse, error)
	Get(ctx context.Context, req *domain.GetReleaseRequest) (*domain.Release, error)
	GetIndexerOptions(ctx context.Context) ([]string, error)
	Stats(ctx context.Context) (*domain.ReleaseStats, error)
	Delete(ctx context.Context, req *domain.DeleteReleaseRequest) error
	CheckSmartEpisodeCanDownload(ctx context.Context, p *domain.SmartEpisodeParams) (bool, error)
	UpdateBaseURL(ctx context.Context, indexer string, oldBaseURL, newBaseURL string) error

	GetActionStatus(ctx context.Context, req *domain.GetReleaseActionStatusRequest) (*domain.ReleaseActionStatus, error)
	StoreReleaseActionStatus(ctx context.Context, status *domain.ReleaseActionStatus) error

	StoreDuplicateProfile(ctx context.Context, profile *domain.DuplicateReleaseProfile) error
	FindDuplicateReleaseProfiles(ctx context.Context) ([]*domain.DuplicateReleaseProfile, error)
	DeleteReleaseProfileDuplicate(ctx context.Context, id int64) error
	CheckIsDuplicateRelease(ctx context.Context, profile *domain.DuplicateReleaseProfile, release *domain.Release) (bool, error)

	releaseCleanupJobRepo
}

type actionService interface {
	Get(ctx context.Context, req *domain.GetActionRequest) (*domain.Action, error)
	FindByFilterID(ctx context.Context, filterID int, active *bool, withClient bool) ([]*domain.Action, error)
	RunAction(ctx context.Context, action *domain.Action, release *domain.Release) (rejections []string, err error)
}

type filterService interface {
	FindByIndexerIdentifier(ctx context.Context, indexer string) ([]*domain.Filter, error)
	CheckFilter(ctx context.Context, f *domain.Filter, release *domain.Release) (bool, error)
}

type indexerService interface {
	GetBy(ctx context.Context, req domain.GetIndexerRequest) (*domain.Indexer, error)
	GetMappedDefinitionByName(name string) (*domain.IndexerDefinition, bool)
}

type schedulerService interface {
	AddJob(job cron.Job, spec string, identifier string) (int, error)
	RemoveJobByIdentifier(id string) error
	GetNextRun(id string) (time.Time, error)
}

type actionClientTypeKey struct {
	Type     domain.ActionType
	ClientID int32
}

// cleanupJobKey creates a unique identifier for controlling cleanup jobs in the scheduler
type cleanupJobKey struct {
	id int
}

// ToString creates a string of the unique id to be used for controlling jobs in the scheduler
func (k cleanupJobKey) ToString() string {
	return fmt.Sprintf("release-cleanup-%d", k.id)
}

type Service struct {
	log         zerolog.Logger
	m           sync.RWMutex
	cleanupJobs map[string]int
	bus         EventBus.Bus

	repo       releaseRepo
	actionSvc  actionService
	filterSvc  filterService
	indexerSvc indexerService
	scheduler  schedulerService
}

func NewService(log zerolog.Logger, repo releaseRepo, actionSvc actionService, filterSvc filterService, indexerSvc indexerService, scheduler schedulerService, bus EventBus.Bus) *Service {
	return &Service{
		log:         log.With().Str("module", "release").Logger(),
		cleanupJobs: map[string]int{},
		bus:         bus,
		repo:        repo,
		actionSvc:   actionSvc,
		filterSvc:   filterSvc,
		indexerSvc:  indexerSvc,
		scheduler:   scheduler,
	}
}

func (s *Service) Find(ctx context.Context, query domain.ReleaseQueryParams) (*domain.FindReleasesResponse, error) {
	return s.repo.Find(ctx, query)
}

func (s *Service) Get(ctx context.Context, req *domain.GetReleaseRequest) (*domain.Release, error) {
	return s.repo.Get(ctx, req)
}

func (s *Service) GetActionStatus(ctx context.Context, req *domain.GetReleaseActionStatusRequest) (*domain.ReleaseActionStatus, error) {
	return s.repo.GetActionStatus(ctx, req)
}

func (s *Service) GetIndexerOptions(ctx context.Context) ([]string, error) {
	return s.repo.GetIndexerOptions(ctx)
}

func (s *Service) Stats(ctx context.Context) (*domain.ReleaseStats, error) {
	return s.repo.Stats(ctx)
}

func (s *Service) Store(ctx context.Context, release *domain.Release) error {
	return s.repo.Store(ctx, release)
}

func (s *Service) Update(ctx context.Context, release *domain.Release) error {
	return s.repo.Update(ctx, release)
}

func (s *Service) StoreReleaseActionStatus(ctx context.Context, status *domain.ReleaseActionStatus) error {
	return s.repo.StoreReleaseActionStatus(ctx, status)
}

func (s *Service) Delete(ctx context.Context, req *domain.DeleteReleaseRequest) error {
	return s.repo.Delete(ctx, req)
}

func (s *Service) FindDuplicateReleaseProfiles(ctx context.Context) ([]*domain.DuplicateReleaseProfile, error) {
	return s.repo.FindDuplicateReleaseProfiles(ctx)
}

func (s *Service) StoreReleaseProfileDuplicate(ctx context.Context, profile *domain.DuplicateReleaseProfile) error {
	return s.repo.StoreDuplicateProfile(ctx, profile)
}

func (s *Service) DeleteReleaseProfileDuplicate(ctx context.Context, id int64) error {
	return s.repo.DeleteReleaseProfileDuplicate(ctx, id)
}

func (s *Service) ListCleanupJobs(ctx context.Context) ([]*domain.ReleaseCleanupJob, error) {
	jobs, err := s.repo.ListCleanupJobs(ctx)
	if err != nil {
		return nil, err
	}

	// Enrich with next run time from scheduler
	for i, job := range jobs {
		if job.Enabled {
			nextRun, err := s.scheduler.GetNextRun(cleanupJobKey{id: job.ID}.ToString())
			if err == nil {
				job.NextRun = nextRun
				jobs[i] = job
			}
		}
	}

	return jobs, nil
}

func (s *Service) GetCleanupJob(ctx context.Context, id int) (*domain.ReleaseCleanupJob, error) {
	return s.repo.FindCleanupJobByID(ctx, id)
}

func (s *Service) StoreCleanupJob(ctx context.Context, job *domain.ReleaseCleanupJob) error {
	// Validate before storing
	if err := job.Validate(); err != nil {
		s.log.Error().Err(err).Msg("cleanup job validation failed")
		return err
	}

	if err := s.repo.StoreCleanupJob(ctx, job); err != nil {
		s.log.Error().Err(err).Interface("data", job).Msg("error storing cleanup job")
		return err
	}

	// Start job if enabled
	if job.Enabled {
		if err := s.startCleanupJob(job); err != nil {
			s.log.Error().Err(err).Str("job", job.Name).Msg("error starting cleanup job")
			return err
		}
	}

	return nil
}

func (s *Service) UpdateCleanupJob(ctx context.Context, job *domain.ReleaseCleanupJob) error {
	// Validate before updating
	if err := job.Validate(); err != nil {
		s.log.Error().Err(err).Msg("cleanup job validation failed")
		return err
	}

	// Get current state before updating
	currentJob, err := s.repo.FindCleanupJobByID(ctx, job.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("error finding cleanup job")
		return err
	}

	if err := s.repo.UpdateCleanupJob(ctx, job); err != nil {
		s.log.Error().Err(err).Msg("error updating cleanup job")
		return err
	}

	// Only restart if job is/was enabled (touching scheduler only when needed)
	if currentJob.Enabled || job.Enabled {
		if err := s.restartCleanupJob(job); err != nil {
			s.log.Error().Err(err).Str("job", job.Name).Msg("error restarting cleanup job")
			return err
		}
	}

	return nil
}

func (s *Service) DeleteCleanupJob(ctx context.Context, id int) error {
	job, err := s.repo.FindCleanupJobByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msg("error finding cleanup job")
		return err
	}

	s.log.Debug().Str("job", job.Name).Msg("deleting cleanup job")

	// Only stop if it's actually running
	if job.Enabled {
		if err := s.stopCleanupJob(id); err != nil {
			s.log.Error().Err(err).Str("job", job.Name).Int("job_id", id).Msg("error stopping cleanup job")
			return err
		}
	}

	// Delete from database
	if err := s.repo.DeleteCleanupJob(ctx, id); err != nil {
		s.log.Error().Err(err).Str("job", job.Name).Msg("error deleting cleanup job")
		return err
	}

	return nil
}

func (s *Service) ToggleCleanupJobEnabled(ctx context.Context, id int, enabled bool) error {
	job, err := s.repo.FindCleanupJobByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msg("error finding cleanup job")
		return err
	}

	// Check if already in desired state
	if job.Enabled == enabled {
		currentStatus := map[bool]string{true: "enabled", false: "disabled"}[enabled]

		s.log.Debug().Str("state", currentStatus).Str("job", job.Name).Msg("cleanup job already in desired state")
		return nil
	}

	// Update database
	if err := s.repo.CleanupJobToggleEnabled(ctx, id, enabled); err != nil {
		s.log.Error().Err(err).Msg("error toggling cleanup job enabled")
		return err
	}

	// Handle scheduler side effects
	if enabled {
		job.Enabled = true
		if err := s.startCleanupJob(job); err != nil {
			s.log.Error().Err(err).Msg("error starting cleanup job")
			return err
		}
		s.log.Debug().Str("job", job.Name).Msg("cleanup job started")
		return nil
	}

	if err := s.stopCleanupJob(id); err != nil {
		s.log.Error().Err(err).Msg("error stopping cleanup job")
		return err
	}
	s.log.Debug().Str("job", job.Name).Msg("cleanup job stopped")
	return nil
}

func (s *Service) ForceRunCleanupJob(ctx context.Context, id int) error {
	job, err := s.repo.FindCleanupJobByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msg("error finding cleanup job")
		return err
	}

	s.log.Info().Str("job", job.Name).Msg("manually triggering cleanup job")

	cleanupJob := NewCleanupJob(s.log.With().Str("job", job.Name).Logger(), s.repo, job)

	cleanupJob.Run()

	return nil
}

func (s *Service) ProcessManual(ctx context.Context, req *domain.ReleaseProcessReq) error {
	// get indexer definition with data
	def, ok := s.indexerSvc.GetMappedDefinitionByName(req.IndexerIdentifier)
	if !ok {
		return domain.ErrIndexerNotFound
	}

	rls := domain.NewRelease(domain.IndexerMinimal{ID: def.ID, Name: def.Name, Identifier: def.Identifier, IdentifierExternal: def.IdentifierExternal})
	rls.TraceID = domain.TraceIDFromCtx(ctx)

	switch req.IndexerImplementation {
	case string(domain.IndexerImplementationIRC):

		// from announce/announce.go
		tmpVars := map[string]string{}
		parseFailed := false

		channelName := def.IRC.Channels[0].Name
		channel, ok := def.IRC.GetChannel(channelName)
		if !ok {
			return errors.New("no channel configured")
		}

		for idx, parseLine := range channel.Parse.Lines {
			match, err := parseLine.ParseLine(tmpVars, req.AnnounceLines[idx], parseLine.Ignore)
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
		if err := channel.Parse.Parse(def, channelName, tmpVars, rls); err != nil {
			return err
		}

	default:
		return errors.New("implementation %q is not supported", req.IndexerImplementation)

	}

	// process
	go s.Process(context.WithoutCancel(ctx), rls)

	return nil
}

func (s *Service) Process(ctx context.Context, release *domain.Release) {
	if release == nil {
		return
	}

	l := s.log.With().Str("trace_id", release.TraceID).Str("indexer", release.Indexer.Identifier).Str("release", release.TorrentName).Logger()

	defer func() {
		if r := recover(); r != nil {
			l.Error().Any("err", r).Str("trace_id", release.TraceID).Msg("recovering from panic in release process")
			//err := errors.New("panic in release process: %s", release.TorrentName)
			return
		}
	}()

	if release.TraceID == "" {
		release.TraceID = domain.TraceIDFromCtx(ctx)
	}

	s.publishEventReleaseNew(release)

	// TODO check in config for "Save all releases"

	// get filters by priority
	filters, err := s.filterSvc.FindByIndexerIdentifier(ctx, release.Indexer.Identifier)
	if err != nil {
		l.Error().Err(err).Msg("release.Process: error finding filters for indexer")
		return
	}

	if len(filters) == 0 {
		l.Debug().Msg("no active filters found for indexer")
		return
	}

	if err := s.processRelease(ctx, release, filters); err != nil {
		l.Error().Err(err).Msg("release.Process: error processing filters for indexer")
		return
	}
}

func (s *Service) processRelease(ctx context.Context, release *domain.Release, filters []*domain.Filter) error {
	defer func(release *domain.Release) {
		err := release.CleanupTemporaryFiles()
		if err != nil {
			s.log.Error().Err(err).Str("trace_id", release.TraceID).Msg("release.Process: error cleaning up temporary files for indexer")
		}
	}(release)

	if err := s.processFilters(ctx, filters, release); err != nil {
		return err
	}

	return nil
}

func (s *Service) processFilters(ctx context.Context, filters []*domain.Filter, release *domain.Release) error {
	// keep track of action clients to avoid sending the same thing all over again
	// save both client type and client id to potentially try another client of same type
	triedActionClients := map[actionClientTypeKey]struct{}{}

	// loop over and check filters
	for _, f := range filters {
		l := s.log.With().Str("trace_id", release.TraceID).Str("indexer", release.Indexer.Identifier).Str("filter", f.Name).Str("release", release.TorrentName).Logger()

		// make the logger available to downstream services via ctx
		subCtx := l.WithContext(ctx)

		// save filter on release
		release.Filter = f
		release.FilterName = f.Name
		release.FilterID = f.ID

		// reset IsDuplicate
		release.IsDuplicate = false
		release.SkipDuplicateProfileID = 0
		release.SkipDuplicateProfileName = ""

		// test filter
		match, err := s.filterSvc.CheckFilter(subCtx, f, release)
		if err != nil {
			l.Error().Err(err).Msg("release.processFilters: error checking filter")
			return err
		}

		if !match || f.RejectReasons.Len() > 0 {
			l.Trace().Str("rejections", f.RejectReasons.String()).Msg("filter rejected release")

			l.Debug().Str("rejections", f.RejectReasons.StringTruncated()).Msg("filter rejected release")
			continue
		}

		l.Info().Msg("filter matched!")

		// found matching filter, lets find the filter actions and attach
		active := true
		actions, err := s.actionSvc.FindByFilterID(subCtx, f.ID, &active, false)
		if err != nil {
			l.Error().Err(err).Msg("release.processFilters: error finding actions for filter")
			return err
		}

		// if no actions, continue to next filter
		if len(actions) == 0 {
			l.Warn().Msg("release.processFilters: no active actions found for filter, trying next one..")
			continue
		}

		// save release here to only save those with rejections from actions instead of all releases
		if release.ID == 0 {
			release.FilterStatus = domain.ReleaseStatusFilterApproved

			if err = s.Store(subCtx, release); err != nil {
				l.Error().Err(err).Interface("release_data", release).Msg("release.processFilters: error writing release to database")
				return err
			}
		}

		var rejections []string

		// run actions (watchFolder, test, exec, qBittorrent, Deluge, arr etc.)
		for idx, act := range actions {
			// only run enabled actions
			if !act.Enabled {
				l.Trace().Str("action", act.Name).Msg("release.processFilters: action is disabled, skipping..")
				continue
			}

			// add action status as pending
			actionStatus := domain.NewReleaseActionStatus(act, release)

			if err := s.StoreReleaseActionStatus(subCtx, actionStatus); err != nil {
				l.Error().Err(err).Msg("release.processFilters: error storing action status for filter")
			}

			if idx == 0 {
				// sleep for the delay period specified in the filter before running actions
				delay := release.Filter.Delay
				if delay > 0 {
					l.Debug().Int("delay", delay).Msg("release.processFilters: delaying processing as specified in the filter")
					time.Sleep(time.Duration(delay) * time.Second)
				}
			}

			l.Trace().Str("action", act.Name).Msg("release.processFilters: run action")

			// keep track of action clients to avoid sending the same thing all over again
			_, tried := triedActionClients[actionClientTypeKey{Type: act.Type, ClientID: act.ClientID}]
			if tried {
				l.Debug().Msg("release.processFilters: action client already tried for this release, skipping..")
				continue
			}

			// run action
			status, err := s.runAction(subCtx, act, release, actionStatus)
			if err != nil {
				l.Error().Err(err).Msg("release.processFilters: error running filter action")
				//continue
			}

			rejections = status.Rejections

			if err := s.StoreReleaseActionStatus(subCtx, status); err != nil {
				l.Error().Err(err).Msg("release.processFilters: error storing action status")
			}

			if len(rejections) > 0 {
				// if we get action rejection, remember which action client it was from
				triedActionClients[actionClientTypeKey{Type: act.Type, ClientID: act.ClientID}] = struct{}{}

				// log something and fire events
				l.Debug().Str("action", act.Name).Str("action_type", string(act.Type)).Strs("rejections", rejections).Msg("action rejected release")
			}

			// if no rejections consider action approved, run next
			continue
		}

		if err = s.Update(subCtx, release); err != nil {
			l.Error().Err(err).Msg("release.processFilters: error updating release")
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

func (s *Service) ProcessMultiple(ctx context.Context, releases []*domain.Release) {
	s.log.Debug().Int("count", len(releases)).Msg("process new releases from feed")

	for _, rls := range releases {
		if rls == nil {
			continue
		}
		s.Process(ctx, rls)
	}
}

func (s *Service) ProcessMultipleFromIndexer(ctx context.Context, releases []*domain.Release, indexer domain.IndexerMinimal) error {
	s.log.Debug().Str("indexer", indexer.Name).Int("count", len(releases)).Msg("process new releases from feed")

	defer func() {
		if r := recover(); r != nil {
			s.log.Error().Any("err", r).Msg("recovering from panic in release process")
			//err := errors.New("panic in release process: %s", release.TorrentName)
			return
		}
	}()

	// get filters by priority
	filters, err := s.filterSvc.FindByIndexerIdentifier(ctx, indexer.Identifier)
	if err != nil {
		s.log.Error().Err(err).Str("indexer", indexer.Name).Msg("release.ProcessMultipleFromIndexer: error finding filters for indexer")
		return err
	}

	// TODO check in config for "Save all releases"

	if len(filters) == 0 {
		// Send RELEASE_NEW notification for ALL incoming releases (before filter checking)
		for _, release := range releases {
			if release == nil {
				continue
			}
			s.publishEventReleaseNew(release)
		}

		s.log.Debug().Str("indexer", indexer.Name).Msg("no active filters found for indexer: skipping filter processing")
		return domain.ErrNoActiveFiltersFoundForIndexer
	}

	for _, release := range releases {
		if release == nil {
			continue
		}

		if release.TraceID == "" {
			release.TraceID = domain.NewTraceID()
		}

		s.publishEventReleaseNew(release)

		if err := s.processRelease(ctx, release, filters); err != nil {
			s.log.Error().Err(err).Str("trace_id", release.TraceID).Str("indexer", indexer.Name).Msg("release.ProcessMultipleFromIndexer: error processing filters for indexer")
			return nil
		}
	}

	return nil
}

func (s *Service) runAction(ctx context.Context, action *domain.Action, release *domain.Release, status *domain.ReleaseActionStatus) (*domain.ReleaseActionStatus, error) {
	// add action status as pending
	//status := domain.NewReleaseActionStatus(action, release)
	//
	//if err := s.StoreReleaseActionStatus(ctx, status); err != nil {
	//	s.log.Error().Err(err).Msgf("release.runAction: error storing action for filter: %s", release.FilterName)
	//}

	rejections, err := s.actionSvc.RunAction(ctx, action, release)
	if err != nil {
		s.log.Error().Err(err).Str("trace_id", release.TraceID).Str("filter", release.FilterName).Msg("release.runAction: error running actions for filter")

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

func (s *Service) retryAction(ctx context.Context, action *domain.Action, release *domain.Release) error {
	l := s.log.With().Str("trace_id", release.TraceID).Str("release", release.TorrentName).Str("filter", release.FilterName).Logger()
	ctx = l.WithContext(ctx)

	// Replaying an action pulls the torrent down again, and writes it to disk if
	// the action uses a path macro. This does not go through processRelease, so
	// it has to clean up after itself.
	defer func(release *domain.Release) {
		if err := release.CleanupTemporaryFiles(); err != nil {
			l.Error().Err(err).Msg("release.retryAction: error cleaning up temporary files")
		}
	}(release)

	// add action status as pending
	status := domain.NewReleaseActionStatus(action, release)

	if err := s.StoreReleaseActionStatus(ctx, status); err != nil {
		l.Error().Err(err).Msg("release.retryAction: error storing action status")
	}

	actionStatus, err := s.runAction(ctx, action, release, status)
	if err != nil {
		l.Error().Err(err).Msg("release.retryAction: error running actions")

		if err := s.StoreReleaseActionStatus(ctx, actionStatus); err != nil {
			l.Error().Err(err).Msg("release.retryAction: error storing action status")
			return err
		}

		return err
	}

	if err := s.StoreReleaseActionStatus(ctx, actionStatus); err != nil {
		l.Error().Err(err).Msg("release.retryAction: error storing action status")
		return err
	}

	return nil
}

func (s *Service) Retry(ctx context.Context, req *domain.ReleaseActionRetryReq) error {
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

	// stored releases have no trace id; reuse the http request id so the
	// replay correlates with the request log
	release.TraceID = domain.TraceIDFromCtx(ctx)

	// run filterAction
	if err := s.retryAction(ctx, filterAction, release); err != nil {
		s.log.Error().Err(err).Str("trace_id", release.TraceID).Str("action", filterAction.Name).Msg("release.Retry: error re-running action")
		return err
	}

	s.log.Info().Str("trace_id", release.TraceID).Str("action", filterAction.Name).Str("release", release.TorrentName).Msg("successfully replayed action for release")

	return nil
}

func (s *Service) publishEventReleaseNew(release *domain.Release) {
	payload := &domain.NotificationPayload{
		Event:          domain.NotificationEventReleaseNew,
		ReleaseName:    release.TorrentName,
		Indexer:        release.Indexer.Name,
		InfoHash:       release.TorrentHash,
		Size:           release.Size,
		Protocol:       release.Protocol,
		Implementation: release.Implementation,
		Timestamp:      time.Now(),
		Release:        release,
	}
	s.bus.Publish(domain.EventNotificationSend, &payload.Event, payload)
}

func (s *Service) startCleanupJob(job *domain.ReleaseCleanupJob) error {
	// If it's not enabled, we should not start it
	if !job.Enabled {
		return errors.New("cleanup job %s not enabled", job.Name)
	}

	// Create the cleanup job instance
	cleanupJob := NewCleanupJob(s.log.With().Str("job", job.Name).Logger(), s.repo, job)

	identifierKey := cleanupJobKey{id: job.ID}.ToString()

	// Schedule job using cron schedule
	id, err := s.scheduler.AddJob(cleanupJob, job.Schedule, identifierKey)
	if err != nil {
		return errors.Wrap(err, "add cleanup job %s failed", identifierKey)
	}

	// Add to job map
	s.m.Lock()
	s.cleanupJobs[identifierKey] = id
	s.m.Unlock()

	s.log.Debug().Str("job", job.Name).Str("job_id", identifierKey).Str("schedule", job.Schedule).Msg("successfully started cleanup job")

	return nil
}

func (s *Service) stopCleanupJob(id int) error {
	identifierKey := cleanupJobKey{id: id}.ToString()

	// Remove job from scheduler
	if err := s.scheduler.RemoveJobByIdentifier(identifierKey); err != nil {
		return errors.Wrap(err, "stop cleanup job failed")
	}

	// Remove from job map
	s.m.Lock()
	delete(s.cleanupJobs, identifierKey)
	s.m.Unlock()

	s.log.Debug().Str("job_id", identifierKey).Msg("stopped cleanup job")

	return nil
}

func (s *Service) restartCleanupJob(job *domain.ReleaseCleanupJob) error {
	s.log.Debug().Str("job", job.Name).Msg("restarting cleanup job")

	// Stop job
	if err := s.stopCleanupJob(job.ID); err != nil {
		s.log.Error().Err(err).Msg("error stopping cleanup job")
		return err
	}

	// Start job if enabled
	if job.Enabled {
		if err := s.startCleanupJob(job); err != nil {
			s.log.Error().Err(err).Msg("error starting cleanup job")
			return err
		}

		s.log.Debug().Str("job", job.Name).Msg("restarted cleanup job")
	}

	return nil
}

func (s *Service) StartCleanupJobs() error {
	ctx := context.TODO()

	// Get all cleanup jobs from database
	jobs, err := s.repo.ListCleanupJobs(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("error finding cleanup jobs")
		return err
	}

	return s.startCleanupJobs(jobs)
}

func (s *Service) startCleanupJobs(jobs []*domain.ReleaseCleanupJob) error {
	if len(jobs) == 0 {
		s.log.Debug().Msg("found 0 cleanup jobs to start")
		return nil
	}

	s.log.Debug().Int("count", len(jobs)).Msg("starting cleanup jobs")

	// Start in background to not block startup
	go func(jobs []*domain.ReleaseCleanupJob) {
		for _, job := range jobs {
			if !job.Enabled {
				s.log.Trace().Str("job", job.Name).Msg("cleanup job disabled, skipping...")
				continue
			}

			if err := s.startCleanupJob(job); err != nil {
				s.log.Error().Err(err).Str("job", job.Name).Msg("failed to initialize cleanup job")
				continue
			}
		}
	}(jobs)

	return nil
}
