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

func (s *service) StoreReleaseActionStatus(ctx context.Context, actionStatus *domain.ReleaseActionStatus) error {
	return s.repo.StoreReleaseActionStatus(ctx, actionStatus)
}

func (s *service) Delete(ctx context.Context) error {
	return s.repo.Delete(ctx)
}

func (s *service) Process(release *domain.Release) {
	if release == nil {
		return
	}

	defer release.CleanupTemporaryFiles()
	ctx := context.Background()

	// TODO check in config for "Save all releases"
	// TODO cross-seed check
	// TODO dupe checks

	// get filters by priority
	filters, err := s.filterSvc.FindByIndexerIdentifier(release.Indexer)
	if err != nil {
		s.log.Error().Err(err).Msgf("release.Process: error finding filters for indexer: %v", release.Indexer)
		return
	}

	if len(filters) == 0 {
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
			l.Trace().Msgf("release.Process: indexer: %v, filter: %v release: %v, no match. rejections: %v", release.Indexer, release.Filter.Name, release.TorrentName, release.RejectionsString())

			l.Debug().Msgf("release rejected: %v", release.RejectionsString())
			continue
		}

		l.Info().Msgf("Matched '%v' (%v) for %v", release.TorrentName, release.Filter.Name, release.Indexer)

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
			l.Debug().Msgf("Delaying processing of '%v' (%v) for %v by %d seconds as specified in the filter", release.TorrentName, release.Filter.Name, release.Indexer, delay)
			time.Sleep(time.Duration(delay) * time.Second)
		}

		var rejections []string

		// run actions (watchFolder, test, exec, qBittorrent, Deluge, arr etc.)
		for _, a := range release.Filter.Actions {
			// only run enabled actions
			if !a.Enabled {
				l.Trace().Msgf("release.Process: indexer: %v, filter: %v release: %v action '%v' not enabled, skip", release.Indexer, release.Filter.Name, release.TorrentName, a.Name)
				continue
			}

			l.Trace().Msgf("release.Process: indexer: %v, filter: %v release: %v , run action: %v", release.Indexer, release.Filter.Name, release.TorrentName, a.Name)

			// keep track of action clients to avoid sending the same thing all over again
			_, tried := triedActionClients[actionClientTypeKey{Type: a.Type, ClientID: a.ClientID}]
			if tried {
				l.Trace().Msgf("release.Process: indexer: %v, filter: %v release: %v action client already tried, skip", release.Indexer, release.Filter.Name, release.TorrentName)
				continue
			}

			rejections, err = s.actionSvc.RunAction(ctx, a, release)
			if err != nil {
				l.Error().Stack().Err(err).Msgf("release.Process: error running actions for filter: %v", release.Filter.Name)
				continue
			}

			if len(rejections) > 0 {
				// if we get a rejection, remember which action client it was from
				triedActionClients[actionClientTypeKey{Type: a.Type, ClientID: a.ClientID}] = struct{}{}

				// log something and fire events
				l.Debug().Str("action", a.Name).Str("action_type", string(a.Type)).Msgf("release rejected: %v", strings.Join(rejections, ", "))
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
