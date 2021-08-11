package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
	_ "modernc.org/sqlite"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/announce"
	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/download_client"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/http"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/irc"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/internal/server"
)

var (
	cfg config.Cfg
)

func main() {
	var configPath string
	pflag.StringVar(&configPath, "config", "", "path to configuration file")
	pflag.Parse()

	// read config
	cfg = config.Read(configPath)

	// setup logger
	logger.Setup(cfg)

	// if configPath is set then put database inside that path, otherwise create wherever it's run
	var dataSource = database.DataSourceName(configPath, "autobrr.db")

	// open database connection
	db, err := sql.Open("sqlite", dataSource)
	if err != nil {
		log.Fatal().Err(err).Msg("could not open db connection")
	}
	defer db.Close()

	if err = database.Migrate(db); err != nil {
		log.Fatal().Err(err).Msg("could not migrate db")
	}

	// setup repos
	// var announceRepo = database.NewAnnounceRepo(db)
	var (
		actionRepo         = database.NewActionRepo(db)
		downloadClientRepo = database.NewDownloadClientRepo(db)
		filterRepo         = database.NewFilterRepo(db)
		indexerRepo        = database.NewIndexerRepo(db)
		ircRepo            = database.NewIrcRepo(db)
	)

	var (
		downloadClientService = download_client.NewService(downloadClientRepo)
		actionService         = action.NewService(actionRepo, downloadClientService)
		indexerService        = indexer.NewService(indexerRepo)
		filterService         = filter.NewService(filterRepo, actionRepo, indexerService)
		releaseService        = release.NewService(actionService)
		announceService       = announce.NewService(filterService, indexerService, releaseService)
		ircService            = irc.NewService(ircRepo, announceService)
	)

	addr := fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)

	errorChannel := make(chan error)

	go func() {
		httpServer := http.NewServer(addr, cfg.BaseURL, actionService, downloadClientService, filterService, indexerService, ircService)
		errorChannel <- httpServer.Open()
	}()

	srv := server.NewServer(ircService, indexerService)
	srv.Hostname = cfg.Host
	srv.Port = cfg.Port

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	if err := srv.Start(); err != nil {
		log.Fatal().Err(err).Msg("could not start server")
	}

	for sig := range sigCh {
		switch sig {
		case syscall.SIGHUP:
			log.Print("shutting down server")
			os.Exit(1)
		case syscall.SIGINT, syscall.SIGTERM:
			log.Print("shutting down server")
			//srv.Shutdown()
			os.Exit(1)
			return
		}
	}
}
