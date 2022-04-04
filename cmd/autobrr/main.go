package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/asaskevich/EventBus"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/auth"
	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/download_client"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/http"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/irc"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/internal/server"
	"github.com/autobrr/autobrr/internal/user"
)

var (
	cfg     domain.Config
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	var configPath string
	pflag.StringVar(&configPath, "config", "", "path to configuration file")
	pflag.Parse()

	// read config
	cfg = config.Read(configPath)

	// setup server-sent-events
	serverEvents := sse.New()
	serverEvents.AutoReplay = false

	serverEvents.CreateStream("logs")

	// setup internal eventbus
	bus := EventBus.New()

	// setup logger
	logger.Setup(cfg, serverEvents)

	// open database connection
	db, _ := database.NewDB(cfg)
	if err := db.Open(); err != nil {
		log.Fatal().Err(err).Msg("could not open db connection")
	}

	log.Info().Msgf("Starting autobrr")
	log.Info().Msgf("Version: %v", version)
	log.Info().Msgf("Commit: %v", commit)
	log.Info().Msgf("Build date: %v", date)
	log.Info().Msgf("Log-level: %v", cfg.LogLevel)
	log.Info().Msgf("Using database: %v", db.Driver)

	// setup repos
	var (
		actionRepo         = database.NewActionRepo(db)
		downloadClientRepo = database.NewDownloadClientRepo(db)
		filterRepo         = database.NewFilterRepo(db)
		indexerRepo        = database.NewIndexerRepo(db)
		ircRepo            = database.NewIrcRepo(db)
		notificationRepo   = database.NewNotificationRepo(db)
		releaseRepo        = database.NewReleaseRepo(db)
		userRepo           = database.NewUserRepo(db)
	)

	// setup services
	var (
		downloadClientService = download_client.NewService(downloadClientRepo)
		actionService         = action.NewService(actionRepo, downloadClientService, bus)
		apiService            = indexer.NewAPIService()
		indexerService        = indexer.NewService(cfg, indexerRepo, apiService)
		filterService         = filter.NewService(filterRepo, actionRepo, apiService, indexerService)
		releaseService        = release.NewService(releaseRepo, actionService)
		ircService            = irc.NewService(ircRepo, filterService, indexerService, releaseService)
		notificationService   = notification.NewService(notificationRepo)
		userService           = user.NewService(userRepo)
		authService           = auth.NewService(userService)
	)

	// register event subscribers
	events.NewSubscribers(bus, notificationService, releaseService)

	errorChannel := make(chan error)

	go func() {
		httpServer := http.NewServer(cfg, serverEvents, version, commit, date, actionService, authService, downloadClientService, filterService, indexerService, ircService, notificationService, releaseService)
		errorChannel <- httpServer.Open()
	}()

	srv := server.NewServer(ircService, indexerService)
	srv.Hostname = cfg.Host
	srv.Port = cfg.Port

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGTERM)

	if err := srv.Start(); err != nil {
		log.Fatal().Stack().Err(err).Msg("could not start server")
		return
	}

	for sig := range sigCh {
		switch sig {
		case syscall.SIGHUP:
			log.Print("shutting down server sighup")
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
