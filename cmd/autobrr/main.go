// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"
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
	"github.com/autobrr/autobrr/internal/list"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/metrics"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/proxy"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/internal/releasedownload"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/internal/server"
	"github.com/autobrr/autobrr/internal/update"
	"github.com/autobrr/autobrr/internal/user"
	"github.com/autobrr/autobrr/pkg/sqlite3store"

	"github.com/KimMachineGun/automemlimit/memlimit"
	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
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
	var configPath, profilePath string
	pflag.StringVar(&configPath, "config", "", "path to configuration directory")
	pflag.StringVar(&profilePath, "pgo", "", "internal build flag")
	pflag.Parse()

	shutdownFunc, isPGO := pgoRun(profilePath)

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

	// Set GOMEMLIMIT to match the Linux container Memory quota (if any)
	memLimit, err := memlimit.SetGoMemLimitWithOpts(memlimit.WithProvider(memlimit.ApplyFallback(memlimit.FromCgroupHybrid, memlimit.FromSystem)))
	if err != nil {
		log.Error().Err(err).Msg("failed to set GOMEMLIMIT")
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
	db, err := database.NewDB(cfg.Config, log)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize database")
	}
	defer db.Close()

	if err := db.Open(); err != nil {
		log.Fatal().Err(err).Msg("could not open db connection")
	}

	log.Info().Msgf("Starting autobrr")
	log.Info().Msgf("Version: %s", version)
	log.Info().Msgf("Commit: %s", commit)
	log.Info().Msgf("Build date: %s", date)
	log.Info().Msgf("Log-level: %s", cfg.Config.LogLevel)
	log.Info().Msgf("Using database: %s", db.Driver)
	log.Debug().Msgf("GOMEMLIMIT: %d bytes", memLimit)

	// session manager
	sessionManager := scs.New()
	switch db.Driver {
	case database.DriverSQLite:
		sessionManager.Store = sqlite3store.New(db)
	case database.DriverPostgres:
		sessionManager.Store = postgresstore.New(db.Handler)
	}

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
		listRepo           = database.NewListRepo(log, db)
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
		userService           = user.NewService(userRepo)
		authService           = auth.NewService(log, userService)
		proxyService          = proxy.NewService(log, proxyRepo)
		indexerAPIService     = indexer.NewAPIService(log, proxyService)
		downloadService       = releasedownload.NewDownloadService(log, releaseRepo, indexerRepo, proxyService)
		downloadClientService = download_client.NewService(log, downloadClientRepo)
		actionService         = action.NewService(log, actionRepo, downloadClientService, downloadService, bus)
		indexerService        = indexer.NewService(log, cfg.Config, bus, indexerRepo, releaseRepo, indexerAPIService, schedulingService)
		filterService         = filter.NewService(log, filterRepo, actionService, releaseRepo, indexerAPIService, indexerService, downloadService, notificationService)
		releaseService        = release.NewService(log, releaseRepo, actionService, filterService, indexerService, schedulingService, bus)
		ircService            = irc.NewService(log, serverEvents, ircRepo, releaseService, indexerService, notificationService, proxyService)
		feedService           = feed.NewService(log, feedRepo, feedCacheRepo, releaseService, proxyService, schedulingService)
		listService           = list.NewService(log, listRepo, downloadClientService, filterService, schedulingService)
	)

	// register event subscribers
	events.NewSubscribers(log, bus, feedService, notificationService, releaseService)

	errorChannel := make(chan error)

	go func() {
		httpServer := http.NewServer(http.Deps{
			Log:                   log,
			SSE:                   serverEvents,
			DB:                    db,
			Config:                cfg,
			SessionManager:        sessionManager,
			Version:               version,
			Commit:                commit,
			Date:                  date,
			ActionService:         actionService,
			ApiService:            apiService,
			AuthService:           authService,
			DownloadClientService: downloadClientService,
			FilterService:         filterService,
			FeedService:           feedService,
			IndexerService:        indexerService,
			IrcService:            ircService,
			ListService:           listService,
			NotificationService:   notificationService,
			ProxyService:          proxyService,
			ReleaseService:        releaseService,
			UpdateService:         updateService,
		},
		)
		errorChannel <- httpServer.Open()
	}()

	if cfg.Config.MetricsEnabled {
		metricsManager := metrics.NewMetricsManager(version, commit, date, releaseService, ircService, feedService, listService, filterService)

		go func() {
			httpMetricsServer := http.NewMetricsServer(
				log,
				cfg,
				version,
				commit,
				date,
				metricsManager,
			)
			errorChannel <- httpMetricsServer.Open()
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	srv := server.NewServer(log, cfg.Config, ircService, indexerService, feedService, releaseService, listService, schedulingService, updateService)
	if err := srv.Start(); err != nil {
		log.Fatal().Stack().Err(err).Msg("could not start server")
		return
	}

	if isPGO {
		time.Sleep(5 * time.Second)
		sigCh <- syscall.SIGQUIT
	}

	for sig := range sigCh {
		log.Info().Msgf("received signal: %v, shutting down server.", sig)

		srv.Shutdown()

		if err := db.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close the database connection properly")
			shutdownFunc()
			os.Exit(1)
		}
		shutdownFunc()
		os.Exit(0)
	}
}

func pgoRun(file string) (func(), bool) {
	if len(file) == 0 {
		return func() {}, false
	}

	f, err := os.Create(file)
	if err != nil {
		log.Fatalf("could not create CPU profile: %v", err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatalf("could not create CPU profile: %v", err)
	}

	return func() {
		defer f.Close()
		defer pprof.StopCPUProfile()
	}, true
}
