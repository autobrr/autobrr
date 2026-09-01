// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package server

import (
	"context"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
)

type serviceStarter interface {
	Start() error
}

type serviceStopper interface {
	Stop()
}

type schedulerService interface {
	serviceStarter
	serviceStopper
}

type feedService interface {
	serviceStarter
}

type indexerService interface {
	serviceStarter
}

type ircService interface {
	StartHandlers()
	StopHandlers()
}

type listService interface {
	serviceStarter
}

type notificationService interface {
	serviceStarter
}

type releaseService interface {
	StartCleanupJobs() error
}

type updateService interface {
	CheckUpdates(ctx context.Context)
}

type Server struct {
	log    zerolog.Logger
	config *domain.Config

	indexerService      indexerService
	ircService          ircService
	feedService         feedService
	releaseService      releaseService
	scheduler           schedulerService
	listService         listService
	notificationService notificationService
	updateService       updateService

	stopWG sync.WaitGroup
	lock   sync.Mutex
}

func NewServer(log zerolog.Logger, config *domain.Config, ircSvc ircService, indexerSvc indexerService, feedSvc feedService, releaseSvc releaseService, listSvc listService, notifySvc notificationService, scheduler schedulerService, updateSvc updateService) *Server {
	return &Server{
		log:                 log.With().Str("module", "server").Logger(),
		config:              config,
		indexerService:      indexerSvc,
		ircService:          ircSvc,
		feedService:         feedSvc,
		releaseService:      releaseSvc,
		listService:         listSvc,
		notificationService: notifySvc,
		scheduler:           scheduler,
		updateService:       updateSvc,
	}
}

func (s *Server) Start() error {
	go s.checkUpdates()

	// start cron scheduler
	_ = s.scheduler.Start() // #nosec G104


	if err := s.notificationService.Start(); err != nil {
		s.log.Error().Err(err).Msg("failed to start notification service")
	}

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

	// start release cleanup scheduler
	if err := s.releaseService.StartCleanupJobs(); err != nil {
		s.log.Error().Err(err).Msg("Could not start release cleanup scheduler")
	}

	// start lists background updater
	go s.listService.Start()

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
