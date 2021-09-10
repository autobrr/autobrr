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
	// announceID (server:channel:announcer)
	def := s.indexerSvc.GetIndexerByAnnounce(announceID)
	if def == nil {
		log.Debug().Msgf("could not find indexer definition: %v", announceID)
		return nil
	}

	announce := domain.Announce{
		Site: def.Identifier,
		Line: msg,
	}

	// parse lines
	if def.Parse.Type == "single" {
		err := s.parseLineSingle(def, &announce, msg)
		if err != nil {
			log.Debug().Msgf("could not parse single line: %v", msg)
			log.Error().Err(err).Msgf("could not parse single line: %v", msg)
			return err
		}
	}
	// implement multiline parsing

	// find filter
	foundFilter, err := s.filterSvc.FindByIndexerIdentifier(announce)
	if err != nil {
		log.Error().Err(err).Msg("could not find filter")
		return err
	}

	// no filter found, lets return
	if foundFilter == nil {
		log.Trace().Msg("no matching filter found")
		return nil
	}
	announce.Filter = foundFilter

	log.Trace().Msgf("announce: %+v", announce)

	log.Info().Msgf("Matched '%v' (%v) for %v", announce.TorrentName, announce.Filter.Name, announce.Site)

	// match release

	// process release
	go func() {
		err = s.releaseSvc.Process(announce)
		if err != nil {
			log.Error().Err(err).Msgf("could not process release: %+v", announce)
		}
	}()

	return nil
}
