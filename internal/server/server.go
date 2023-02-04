package server

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/autobrr/autobrr/internal/feed"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/irc"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/internal/update"
)

type Server struct {
	log      zerolog.Logger
	Hostname string
	Port     int
	Version  string

	indexerService indexer.Service
	ircService     irc.Service
	feedService    feed.Service
	scheduler      scheduler.Service
	updateService  *update.Service

	stopWG sync.WaitGroup
	lock   sync.Mutex
}

func NewServer(log logger.Logger, version string, ircSvc irc.Service, indexerSvc indexer.Service, feedSvc feed.Service, scheduler scheduler.Service, updateSvc *update.Service) *Server {
	return &Server{
		log:            log.With().Str("module", "server").Logger(),
		Version:        version,
		indexerService: indexerSvc,
		ircService:     ircSvc,
		feedService:    feedSvc,
		scheduler:      scheduler,
		updateService:  updateSvc,
	}
}

func (s *Server) Start() error {
	s.log.Info().Msgf("Starting server. Listening on %v:%v", s.Hostname, s.Port)

	go s.checkUpdates()

	// start cron scheduler
	s.scheduler.Start()

	// instantiate indexers
	if err := s.indexerService.Start(); err != nil {
		s.log.Error().Err(err).Msg("Could not start indexer service")
		return err
	}

	// instantiate and start irc networks
	s.ircService.StartHandlers()

	// start torznab feeds
	if err := s.feedService.Start(); err != nil {
		s.log.Error().Err(err).Msg("Could not start feed service")
	}

	return nil
}

func (s *Server) Shutdown() {
	s.log.Info().Msg("Shutting down server")

	// stop all irc handlers
	s.ircService.StopHandlers()

	// stop cron scheduler
	s.scheduler.Stop()
}

func (s *Server) checkUpdates() {
	time.Sleep(1 * time.Second)

	s.updateService.CheckUpdates(context.Background())
}
