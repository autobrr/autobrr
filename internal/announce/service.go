package announce

import (
	"context"
	"strings"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/release"

	"github.com/rs/zerolog/log"
)

type Service interface {
	Process(release *domain.Release)
}

type service struct {
	actionSvc  action.Service
	filterSvc  filter.Service
	releaseSvc release.Service
}

type actionClientTypeKey struct {
	Type     domain.ActionType
	ClientID int32
}

func NewService(actionSvc action.Service, filterSvc filter.Service, releaseSvc release.Service) Service {
	return &service{
		actionSvc:  actionSvc,
		filterSvc:  filterSvc,
		releaseSvc: releaseSvc,
	}
}

func (s *service) Process(release *domain.Release) {
	// TODO check in config for "Save all releases"
	// TODO cross-seed check
	// TODO dupe checks

	// get filters by priority
	filters, err := s.filterSvc.FindByIndexerIdentifier(release.Indexer)
	if err != nil {
		log.Error().Err(err).Msgf("announce.Service.Process: error finding filters for indexer: %v", release.Indexer)
		return
	}

	// keep track of action clients to avoid sending the same thing all over again
	// save both client type and client id to potentially try another client of same type
	triedActionClients := map[actionClientTypeKey]struct{}{}

	// loop over and check filters
	for _, f := range filters {
		// save filter on release
		release.Filter = &f
		release.FilterName = f.Name
		release.FilterID = f.ID

		// TODO filter limit checks

		// test filter
		match, err := s.filterSvc.CheckFilter(f, release)
		if err != nil {
			log.Error().Err(err).Msg("announce.Service.Process: could not find filter")
			return
		}

		if !match {
			log.Trace().Msgf("announce.Service.Process: indexer: %v, filter: %v release: %v, no match", release.Indexer, release.Filter.Name, release.TorrentName)
			continue
		}

		log.Info().Msgf("Matched '%v' (%v) for %v", release.TorrentName, release.Filter.Name, release.Indexer)

		// save release here to only save those with rejections from actions instead of all releases
		if release.ID == 0 {
			release.FilterStatus = domain.ReleaseStatusFilterApproved
			err = s.releaseSvc.Store(context.Background(), release)
			if err != nil {
				log.Error().Err(err).Msgf("announce.Service.Process: error writing release to database: %+v", release)
				return
			}
		}

		var rejections []string

		// run actions (watchFolder, test, exec, qBittorrent, Deluge, arr etc.)
		for _, a := range release.Filter.Actions {
			// only run enabled actions
			if !a.Enabled {
				log.Trace().Msgf("announce.Service.Process: indexer: %v, filter: %v release: %v action '%v' not enabled, skip", release.Indexer, release.Filter.Name, release.TorrentName, a.Name)
				continue
			}

			log.Trace().Msgf("announce.Service.Process: indexer: %v, filter: %v release: %v , run action: %v", release.Indexer, release.Filter.Name, release.TorrentName, a.Name)

			// keep track of action clients to avoid sending the same thing all over again
			_, tried := triedActionClients[actionClientTypeKey{Type: a.Type, ClientID: a.ClientID}]
			if tried {
				log.Trace().Msgf("announce.Service.Process: indexer: %v, filter: %v release: %v action client already tried, skip", release.Indexer, release.Filter.Name, release.TorrentName)
				continue
			}

			rejections, err = s.actionSvc.RunAction(a, *release)
			if err != nil {
				log.Error().Stack().Err(err).Msgf("announce.Service.Process: error running actions for filter: %v", release.Filter.Name)
				continue
			}

			if len(rejections) > 0 {
				// if we get a rejection, remember which action client it was from
				triedActionClients[actionClientTypeKey{Type: a.Type, ClientID: a.ClientID}] = struct{}{}

				// log something and fire events
				log.Debug().Msgf("announce.Service.Process: indexer: %v, filter: %v release: %v, rejected: %v", release.Indexer, release.Filter.Name, release.TorrentName, strings.Join(rejections, ", "))
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
