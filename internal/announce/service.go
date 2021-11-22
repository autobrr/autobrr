package announce

import (
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/rs/zerolog/log"
)

type Service interface {
	Parse(announceID string, msg string) error
}

type service struct {
	filterSvc  filter.Service
	indexerSvc indexer.Service
	releaseSvc release.Service
	queues     map[string]chan string
}

func NewService(filterService filter.Service, indexerSvc indexer.Service, releaseService release.Service) Service {

	//queues := make(map[string]chan string)
	//for _, channel := range tinfo {
	//
	//}

	return &service{
		filterSvc:  filterService,
		indexerSvc: indexerSvc,
		releaseSvc: releaseService,
	}
}

// Parse announce line
func (s *service) Parse(announceID string, msg string) error {
	// make simpler by injecting indexer, or indexerdefinitions

	// announceID (server:channel:announcer)
	definition := s.indexerSvc.GetIndexerByAnnounce(announceID)
	if definition == nil {
		log.Debug().Msgf("could not find indexer definition: %v", announceID)
		return nil
	}

	newRelease, err := domain.NewRelease(definition.Identifier, msg)
	if err != nil {
		log.Error().Err(err).Msg("could not create new release")
		return err
	}

	// parse lines
	if definition.Parse.Type == "single" {
		err = s.parseLineSingle(definition, newRelease, msg)
		if err != nil {
			log.Error().Err(err).Msgf("could not parse single line: %v", msg)
			return err
		}
	}

	// TODO implement multiline parsing

	filterOK, foundFilter, err := s.filterSvc.FindAndCheckFilters(newRelease)
	if err != nil {
		log.Error().Err(err).Msg("could not find filter")
		return err
	}

	// no foundFilter found, lets return
	if !filterOK || foundFilter == nil {
		log.Trace().Msg("no matching filter found")

		// TODO check in config for "Save all releases"
		// Save as rejected
		newRelease.FilterStatus = domain.ReleaseStatusFilterRejected
		err = s.releaseSvc.Store(newRelease)
		if err != nil {
			log.Error().Err(err).Msgf("error writing release to database: %+v", newRelease)
			return nil
		}
		return nil
	}

	// save release
	newRelease.Filter = foundFilter
	newRelease.FilterName = foundFilter.Name
	newRelease.FilterID = foundFilter.ID

	newRelease.FilterStatus = domain.ReleaseStatusFilterApproved
	err = s.releaseSvc.Store(newRelease)
	if err != nil {
		log.Error().Err(err).Msgf("error writing release to database: %+v", newRelease)
		return nil
	}

	log.Trace().Msgf("release: %+v", newRelease)

	log.Info().Msgf("Matched '%v' (%v) for %v", newRelease.TorrentName, newRelease.Filter.Name, newRelease.Indexer)

	// process release
	go func() {
		// TODO Pointer??
		err = s.releaseSvc.Process(*newRelease)
		if err != nil {
			log.Error().Err(err).Msgf("could not process release: %+v", newRelease)
		}
	}()

	return nil
}
