package server

import (
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/irc"
)

type Server struct {
	Hostname string
	Port     int

	indexerService indexer.Service
	ircService     irc.Service

	stopWG sync.WaitGroup
	lock   sync.Mutex
}

func NewServer(ircSvc irc.Service, indexerSvc indexer.Service) *Server {
	return &Server{
		indexerService: indexerSvc,
		ircService:     ircSvc,
	}
}

func (s *Server) Start() error {
	log.Info().Msgf("Starting server. Listening on %v:%v", s.Hostname, s.Port)

	// instantiate indexers
	err := s.indexerService.Start()
	if err != nil {
		return err
	}

	// instantiate and start irc networks
	s.ircService.StartHandlers()

	return nil
}

func (s *Server) Shutdown() {
	log.Info().Msg("Shutting down server")

	// stop all irc handlers
	s.ircService.StopHandlers()
}
