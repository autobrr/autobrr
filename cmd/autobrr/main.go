package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/asaskevich/EventBus"
	"github.com/r3labs/sse/v2"
	"github.com/spf13/pflag"

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
	"github.com/autobrr/autobrr/internal/user"

	_ "time/tzdata"
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
	serverEvents.AutoReplay = false
	serverEvents.CreateStream("logs")

	// register SSE hook on logger
	log.RegisterSSEHook(serverEvents)

	// setup internal eventbus
	bus := EventBus.New()

	// open database connection
	db, _ := database.NewDB(cfg.Config, log)
	if err := db.Open(); err != nil {
		log.Fatal().Err(err).Msg("could not open db connection")
	}

	log.Info().Msgf("Starting autobrr")
	log.Info().Msgf("Version: %v", version)
	log.Info().Msgf("Commit: %v", commit)
	log.Info().Msgf("Build date: %v", date)
	log.Info().Msgf("Log-level: %v", cfg.Config.LogLevel)
	log.Info().Msgf("Using database: %v", db.Driver)

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
		schedulingService     = scheduler.NewService(log, version, notificationService)
		indexerAPIService     = indexer.NewAPIService(log)
		userService           = user.NewService(userRepo)
		authService           = auth.NewService(log, userService)
		downloadClientService = download_client.NewService(log, downloadClientRepo)
		actionService         = action.NewService(log, actionRepo, downloadClientService, bus)
		indexerService        = indexer.NewService(log, cfg.Config, indexerRepo, indexerAPIService, schedulingService)
		filterService         = filter.NewService(log, filterRepo, actionRepo, releaseRepo, indexerAPIService, indexerService)
		releaseService        = release.NewService(log, releaseRepo, actionService, filterService)
		ircService            = irc.NewService(log, ircRepo, releaseService, indexerService, notificationService)
		feedService           = feed.NewService(log, feedRepo, feedCacheRepo, releaseService, schedulingService)
	)

	// register event subscribers
	events.NewSubscribers(log, bus, notificationService, releaseService)

	errorChannel := make(chan error)

	go func() {
		httpServer := http.NewServer(
			cfg.Config,
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
		)
		errorChannel <- httpServer.Open()
	}()

	srv := server.NewServer(log, ircService, indexerService, feedService, schedulingService)
	srv.Hostname = cfg.Config.Host
	srv.Port = cfg.Config.Port

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM)

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
