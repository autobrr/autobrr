// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/api"
	"github.com/autobrr/autobrr/internal/auth"
	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/download_client"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/internal/feed"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/http"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/irc"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/internal/server"
	"github.com/autobrr/autobrr/internal/update"
	"github.com/autobrr/autobrr/internal/user"

	"github.com/asaskevich/EventBus"
	"github.com/r3labs/sse/v2"
	"github.com/spf13/pflag"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	var configPath string
	pflag.StringVar(&configPath, "config", "", "path to configuration file")
	pflag.Parse()

	// read config
	cfg := config.New(configPath, version)

	// init new logger
	log := logger.New(cfg.Config)

	// init dynamic config
	cfg.DynamicReload(log)

	// setup server-sent-events
	serverEvents := sse.New()
	serverEvents.CreateStreamWithOpts("logs", sse.StreamOpts{MaxEntries: 1000, AutoReplay: true})

	// register SSE hook on logger
	log.RegisterSSEWriter(serverEvents)

	// setup internal eventbus
	bus := EventBus.New()

	// open database connection
	db, _ := database.NewDB(cfg.Config, log)
	if err := db.Open(); err != nil {
		log.Fatal().Err(err).Msg("could not open db connection")
	}

	log.Info().Msgf("Starting autobrr")
	log.Info().Msgf("Version: %s", version)
	log.Info().Msgf("Commit: %s", commit)
	log.Info().Msgf("Build date: %s", date)
	log.Info().Msgf("Log-level: %s", cfg.Config.LogLevel)
	log.Info().Msgf("Using database: %s", db.Driver)

	// setup repos
	var (
		apikeyRepo         = database.NewAPIRepo(log, db)
		downloadClientRepo = database.NewDownloadClientRepo(log, db)
		actionRepo         = database.NewActionRepo(log, db, downloadClientRepo)
		filterRepo         = database.NewFilterRepo(log, db)
		feedRepo           = database.NewFeedRepo(log, db)
		feedCacheRepo      = database.NewFeedCacheRepo(log, db)
		indexerRepo        = database.NewIndexerRepo(log, db)
		ircRepo            = database.NewIrcRepo(log, db)
		notificationRepo   = database.NewNotificationRepo(log, db)
		releaseRepo        = database.NewReleaseRepo(log, db)
		userRepo           = database.NewUserRepo(log, db)
	)

	// setup services
	var (
		apiService            = api.NewService(log, apikeyRepo)
		notificationService   = notification.NewService(log, notificationRepo)
		updateService         = update.NewUpdate(log, cfg.Config)
		schedulingService     = scheduler.NewService(log, cfg.Config, notificationService, updateService)
		indexerAPIService     = indexer.NewAPIService(log)
		userService           = user.NewService(userRepo)
		authService           = auth.NewService(log, userService)
		downloadClientService = download_client.NewService(log, downloadClientRepo)
		actionService         = action.NewService(log, actionRepo, downloadClientService, bus)
		indexerService        = indexer.NewService(log, cfg.Config, indexerRepo, indexerAPIService, schedulingService)
		filterService         = filter.NewService(log, filterRepo, actionRepo, releaseRepo, indexerAPIService, indexerService)
		releaseService        = release.NewService(log, releaseRepo, actionService, filterService)
		ircService            = irc.NewService(log, serverEvents, ircRepo, releaseService, indexerService, notificationService)
		feedService           = feed.NewService(log, feedRepo, feedCacheRepo, releaseService, schedulingService)
	)

	// register event subscribers
	events.NewSubscribers(log, bus, notificationService, releaseService)

	errorChannel := make(chan error)

	go func() {
		httpServer := http.NewServer(
			log,
			cfg,
			serverEvents,
			db,
			version,
			commit,
			date,
			actionService,
			apiService,
			authService,
			downloadClientService,
			filterService,
			feedService,
			indexerService,
			ircService,
			notificationService,
			releaseService,
			updateService,
		)
		errorChannel <- httpServer.Open()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM)

	srv := server.NewServer(log, cfg.Config, ircService, indexerService, feedService, schedulingService, updateService)
	if err := srv.Start(); err != nil {
		log.Fatal().Stack().Err(err).Msg("could not start server")
		return
	}

	for sig := range sigCh {
		switch sig {
		case syscall.SIGHUP:
			log.Log().Msg("shutting down server sighup")
			srv.Shutdown()
			db.Close()
			os.Exit(1)
		case syscall.SIGINT, syscall.SIGQUIT:
			srv.Shutdown()
			db.Close()
			os.Exit(1)
		case syscall.SIGKILL, syscall.SIGTERM:
			srv.Shutdown()
			db.Close()
			os.Exit(1)
		}
	}
}
