// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"context"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/feed"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/irc"
	"github.com/autobrr/autobrr/internal/list"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/internal/update"

	"github.com/rs/zerolog"
)

type Server struct {
	log    zerolog.Logger
	config *domain.Config

	indexerService indexer.Service
	ircService     irc.Service
	feedService    feed.Service
	scheduler      scheduler.Service
	listService    list.Service
	updateService  *update.Service

	stopWG sync.WaitGroup
	lock   sync.Mutex
}

func NewServer(log logger.Logger, config *domain.Config, ircSvc irc.Service, indexerSvc indexer.Service, feedSvc feed.Service, listSvc list.Service, scheduler scheduler.Service, updateSvc *update.Service) *Server {
	return &Server{
		log:            log.With().Str("module", "server").Logger(),
		config:         config,
		indexerService: indexerSvc,
		ircService:     ircSvc,
		feedService:    feedSvc,
		listService:    listSvc,
		scheduler:      scheduler,
		updateService:  updateSvc,
	}
}

func (s *Server) Start() error {
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

	// start lists background updater
	s.listService.Start()

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
	if s.config.CheckForUpdates {
		time.Sleep(1 * time.Second)

		s.updateService.CheckUpdates(context.Background())
	}
}
