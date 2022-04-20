package server

import (
	"sync"

	"github.com/autobrr/autobrr/internal/feed"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/irc"

	"github.com/rs/zerolog/log"
)

type Server struct {
	Hostname string
	Port     int

	indexerService indexer.Service
	ircService     irc.Service
	feedService    feed.Service

	stopWG sync.WaitGroup
	lock   sync.Mutex
}

func NewServer(ircSvc irc.Service, indexerSvc indexer.Service, feedSvc feed.Service) *Server {
	return &Server{
		indexerService: indexerSvc,
		ircService:     ircSvc,
		feedService:    feedSvc,
	}
}

func (s *Server) Start() error {
	log.Info().Msgf("Starting server. Listening on %v:%v", s.Hostname, s.Port)

	// instantiate indexers
	if err := s.indexerService.Start(); err != nil {
		log.Error().Err(err).Msg("Could not start indexer service")
		return err
	}

	// instantiate and start irc networks
	s.ircService.StartHandlers()

	// start torznab feeds
	if err := s.feedService.Start(); err != nil {
		log.Error().Err(err).Msg("Could not start feed service")
	}

	return nil
}

func (s *Server) Shutdown() {
	log.Info().Msg("Shutting down server")

	// stop all irc handlers
	s.ircService.StopHandlers()

	// stop feed service and cron jobs
	s.feedService.Stop()
}
