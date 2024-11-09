// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
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
	"github.com/autobrr/autobrr/internal/diagnostics"
	"github.com/autobrr/autobrr/internal/download_client"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/internal/feed"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/http"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/irc"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/proxy"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/internal/releasedownload"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/internal/server"
	"github.com/autobrr/autobrr/internal/update"
	"github.com/autobrr/autobrr/internal/user"

	"github.com/asaskevich/EventBus"
	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"go.uber.org/automaxprocs/maxprocs"
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

	// Set GOMAXPROCS to match the Linux container CPU quota (if any)
	undo, err := maxprocs.Set(maxprocs.Logger(zstdlog.NewStdLoggerWithLevel(log.With().Logger(), zerolog.InfoLevel).Printf))
	defer undo()
	if err != nil {
		log.Error().Err(err).Msg("failed to set GOMAXPROCS")
	}

	// init dynamic config
	cfg.DynamicReload(log)

	diagnostics.SetupProfiling(cfg.Config.ProfilingEnabled, cfg.Config.ProfilingHost, cfg.Config.ProfilingPort)

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
		proxyRepo          = database.NewProxyRepo(log, db)
	)

	// setup services
	var (
		apiService            = api.NewService(log, apikeyRepo)
		updateService         = update.NewUpdate(log, cfg.Config)
		notificationService   = notification.NewService(log, notificationRepo)
		schedulingService     = scheduler.NewService(log, cfg.Config, notificationService, updateService)
		indexerAPIService     = indexer.NewAPIService(log)
		userService           = user.NewService(userRepo)
		authService           = auth.NewService(log, userService)
		proxyService          = proxy.NewService(log, proxyRepo)
		downloadService       = releasedownload.NewDownloadService(log, releaseRepo, indexerRepo, proxyService)
		downloadClientService = download_client.NewService(log, downloadClientRepo)
		actionService         = action.NewService(log, actionRepo, downloadClientService, downloadService, bus)
		indexerService        = indexer.NewService(log, cfg.Config, bus, indexerRepo, releaseRepo, indexerAPIService, schedulingService)
		filterService         = filter.NewService(log, filterRepo, actionService, releaseRepo, indexerAPIService, indexerService, downloadService)
		releaseService        = release.NewService(log, releaseRepo, actionService, filterService, indexerService)
		ircService            = irc.NewService(log, serverEvents, ircRepo, releaseService, indexerService, notificationService, proxyService)
		feedService           = feed.NewService(log, feedRepo, feedCacheRepo, releaseService, proxyService, schedulingService)
	)

	// register event subscribers
	events.NewSubscribers(log, bus, feedService, notificationService, releaseService)

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
			proxyService,
			releaseService,
			updateService,
		)
		errorChannel <- httpServer.Open()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	srv := server.NewServer(log, cfg.Config, ircService, indexerService, feedService, schedulingService, updateService)
	if err := srv.Start(); err != nil {
		log.Fatal().Stack().Err(err).Msg("could not start server")
		return
	}

	for sig := range sigCh {
		log.Info().Msgf("received signal: %v, shutting down server.", sig)

		srv.Shutdown()

		if err := db.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close the database connection properly")
			os.Exit(1)
		}
		os.Exit(0)
	}
}
