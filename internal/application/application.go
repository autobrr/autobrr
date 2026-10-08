// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package application

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/alert"
	"github.com/autobrr/autobrr/internal/api"
	"github.com/autobrr/autobrr/internal/auth"
	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/diagnostics"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/downloader"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/internal/feed"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/http"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/irc"
	"github.com/autobrr/autobrr/internal/list"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/meta"
	"github.com/autobrr/autobrr/internal/metrics"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/proxy"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/internal/update"
	"github.com/autobrr/autobrr/internal/user"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/featureflags"
	"github.com/autobrr/autobrr/pkg/sqlite3store"
	"github.com/autobrr/autobrr/pkg/version"

	"github.com/KimMachineGun/automemlimit/memlimit"
	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
	"go.uber.org/automaxprocs/maxprocs"
)

func init() {
	featureflags.Register(domain.IRCFuzzyAnnouncer, false)
}

// App owns the lifecycle of every service, from config load to shutdown.
type App struct {
	log           zerolog.Logger
	cfg           *config.AppConfig
	sse           *sse.Server
	eventBus      *events.EventBus
	updateService *update.Service

	db            *database.DB
	httpServer    *http.Server
	metricsServer *http.MetricsServer
	profiler      *diagnostics.Profiler

	actionService       *action.Service
	alertService        *alert.Service
	authService         *auth.Service
	apiService          *api.Service
	downloaderService   *downloader.Service
	indexerService      *indexer.Service
	indexerAPIService   *indexer.APIService
	ircService          *irc.Service
	feedService         *feed.Service
	filterService       *filter.Service
	notificationService *notification.Service
	releaseService      *release.Service
	rlsDownloadService  *release.DownloadService
	scheduler           *scheduler.Service
	listService         *list.Service
	metricsManager      *metrics.Manager
	proxyService        *proxy.Service
	userService         *user.Service
}

// New loads the config and builds the parts that need no database. They exist
// before Run so a host such as the tray can subscribe to events and check for
// updates while the rest of the application starts.
func New(configPath string) *App {
	cfg := config.New(configPath)

	serverEvents := sse.New()
	serverEvents.CreateStreamWithOpts(logger.StreamLogs, sse.StreamOpts{MaxEntries: 1000, AutoReplay: true})
	serverEvents.CreateStreamWithOpts("irc", sse.StreamOpts{MaxEntries: 0, AutoReplay: false, AutoStream: true})
	serverEvents.CreateStreamWithOpts(notification.InboxStreamKey, sse.StreamOpts{MaxEntries: 0, AutoReplay: false})

	log := logger.New(cfg.Config, serverEvents)

	cfg.DynamicReload(log)

	return &App{
		log:           log,
		cfg:           cfg,
		sse:           serverEvents,
		eventBus:      events.NewEventBus(log),
		updateService: update.NewUpdate(log, cfg.Config),
	}
}

// Run starts the application and blocks until ctx is cancelled or one of its
// servers exits unexpectedly, then shuts everything down. Cancelling ctx is a
// clean shutdown and returns nil.
func (app *App) Run(ctx context.Context) error {
	if err := app.setup(ctx); err != nil {
		return err
	}

	defer func() {
		if err := app.db.Close(); err != nil {
			app.log.Error().Err(err).Msg("could not close database")
		}
	}()

	runCtx, fail := context.WithCancelCause(ctx)
	defer fail(nil)

	started, err := app.startComponents(runCtx, app.components(fail))
	if err != nil {
		app.stopComponents(started)

		return err
	}

	<-runCtx.Done()

	app.log.Info().Msg("shutting down")

	app.stopComponents(started)

	if ctx.Err() != nil {
		return nil
	}

	return context.Cause(runCtx)
}

func (app *App) components(fail context.CancelCauseFunc) []component {
	var components []component

	if app.profiler != nil {
		components = append(components, component{
			name:     "profiler",
			optional: true,
			start: func(context.Context) error {
				if err := app.profiler.Listen(); err != nil {
					return err
				}

				serve(app.profiler.Serve, func(err error) {
					app.log.Error().Err(err).Str("component", "profiler").Msg("server stopped unexpectedly")
				})

				return nil
			},
			stop: app.profiler.Shutdown,
		})
	}

	components = append(components,
		component{
			name: "http",
			start: func(context.Context) error {
				if err := app.httpServer.Listen(); err != nil {
					return err
				}

				serve(app.httpServer.Serve, func(err error) {
					fail(errors.Wrap(err, "http server stopped unexpectedly"))
				})

				return nil
			},
			stop: app.httpServer.Shutdown,
		},
		component{
			name: "scheduler",
			start: func(context.Context) error {
				return app.scheduler.Start()
			},
			stop: func(context.Context) error {
				app.scheduler.Stop()

				return nil
			},
		},
		component{
			name:     "notifications",
			optional: true,
			start: func(context.Context) error {
				return app.notificationService.Start()
			},
		},
		component{
			name: "indexers",
			start: func(context.Context) error {
				return app.indexerService.Start()
			},
		},
		component{
			name: "irc",
			start: func(context.Context) error {
				app.ircService.StartHandlers()

				return nil
			},
			stop: func(context.Context) error {
				app.ircService.StopHandlers()

				return nil
			},
		},
		component{
			name:     "feeds",
			optional: true,
			start: func(context.Context) error {
				return app.feedService.Start()
			},
		},
		component{
			name:     "release cleanup",
			optional: true,
			start: func(context.Context) error {
				return app.releaseService.StartCleanupJobs()
			},
		},
		component{
			name: "lists",
			start: func(context.Context) error {
				go func() {
					if err := app.listService.Start(); err != nil {
						app.log.Error().Err(err).Msg("could not start list service")
					}
				}()

				return nil
			},
		},
	)

	if app.metricsServer != nil {
		components = append(components, component{
			name:     "metrics",
			optional: true,
			start: func(context.Context) error {
				if err := app.metricsServer.Listen(); err != nil {
					return err
				}

				serve(app.metricsServer.Serve, func(err error) {
					app.log.Error().Err(err).Str("component", "metrics").Msg("server stopped unexpectedly")
				})

				return nil
			},
			stop: app.metricsServer.Shutdown,
		})
	}

	if app.cfg.Config.CheckForUpdates {
		components = append(components, component{
			name: "update check",
			start: func(ctx context.Context) error {
				go app.checkUpdates(ctx)

				return nil
			},
		})
	}

	return components
}

// Config returns the loaded configuration.
func (app *App) Config() *domain.Config {
	return app.cfg.Config
}

// Log returns the application logger.
func (app *App) Log() *zerolog.Logger {
	return &app.log
}

// CheckUpdateAvailable checks for a newer release. It is safe to call before Run.
func (app *App) CheckUpdateAvailable(ctx context.Context) (*version.Release, error) {
	return app.updateService.CheckUpdateAvailable(ctx)
}

// OnAppUpdate subscribes to new-release events and returns the unsubscribe func.
func (app *App) OnAppUpdate(handler func(context.Context, events.AppUpdateEvent) error) func() {
	return app.eventBus.OnAppUpdate(handler)
}

func (app *App) setupResourceLimits() {
	// the returned undo is dropped on purpose, the limit holds for the process lifetime
	if _, err := maxprocs.Set(maxprocs.Logger(zstdlog.NewStdLoggerWithLevel(app.log.With().Logger(), zerolog.InfoLevel).Printf)); err != nil {
		app.log.Error().Err(err).Msg("failed to set GOMAXPROCS")
	}

	memLimit, err := memlimit.Set(memlimit.WithProvider(memlimit.ApplyFallback(memlimit.FromCgroup, memlimit.FromSystem)))
	if err != nil {
		app.log.Error().Err(err).Msg("failed to set GOMEMLIMIT")
	}

	app.log.Debug().Int64("gomemlimit_bytes", memLimit).Msg("memory limit configured")
}

func (app *App) setup(ctx context.Context) error {
	app.setupResourceLimits()

	oidcService := auth.NewOIDCService(app.log, app.cfg.Config)
	if err := oidcService.Discover(ctx); err != nil {
		return errors.Wrap(err, "could not discover OIDC service")
	}

	db, err := database.NewDB(app.cfg.Config, app.log)
	if err != nil {
		return errors.Wrap(err, "could not init database")
	}

	if err := db.Open(); err != nil {
		return errors.Wrap(err, "could not open database")
	}

	app.db = db

	app.log.Info().
		Str("version", meta.GetVersion()).
		Str("commit", meta.GetCommit()).
		Str("build_date", meta.GetDate()).
		Str("log_level", app.cfg.Config.LogLevel).
		Str("database", db.Driver).
		Msg("starting autobrr")

	sessionManager := scs.New()
	switch db.Driver {
	case database.DriverSQLite:
		sessionManager.Store = sqlite3store.New(db, sqlite3store.WithLogger(app.log.With().Str("module", "session-store").Logger()))
	case database.DriverPostgres:
		sessionManager.Store = postgresstore.New(db.Handler)
	}

	var (
		apikeyRepo       = database.NewAPIRepo(app.log, db)
		downloaderRepo   = database.NewDownloaderRepo(app.log, db)
		actionRepo       = database.NewActionRepo(app.log, db)
		filterRepo       = database.NewFilterRepo(app.log, db)
		feedRepo         = database.NewFeedRepo(app.log, db)
		feedCacheRepo    = database.NewFeedCacheRepo(app.log, db)
		indexerRepo      = database.NewIndexerRepo(app.log, db)
		ircRepo          = database.NewIrcRepo(app.log, db)
		listRepo         = database.NewListRepo(app.log, db)
		notificationRepo = database.NewNotificationRepo(app.log, db)
		inboxRepo        = database.NewNotificationInboxRepo(app.log, db)
		releaseRepo      = database.NewReleaseRepo(app.log, db)
		userRepo         = database.NewUserRepo(app.log, db)
		proxyRepo        = database.NewProxyRepo(app.log, db)
	)

	app.apiService = api.NewService(app.log, apikeyRepo)
	app.scheduler = scheduler.NewService(app.log, app.eventBus, app.cfg.Config, app.updateService)
	app.notificationService = notification.NewService(app.log, app.eventBus, app.sse, notificationRepo, inboxRepo, app.scheduler)
	app.userService = user.NewService(userRepo)
	app.authService = auth.NewService(app.log, app.userService)
	app.proxyService = proxy.NewService(app.log, app.eventBus, proxyRepo)
	app.indexerAPIService = indexer.NewAPIService(app.log, app.proxyService)
	app.rlsDownloadService = release.NewDownloadService(app.log, indexerRepo, app.proxyService)
	app.downloaderService = downloader.NewService(app.log, downloaderRepo)
	app.actionService = action.NewService(app.log, app.eventBus, actionRepo, app.downloaderService, app.rlsDownloadService)
	app.indexerService = indexer.NewService(app.log, app.eventBus, app.cfg.Config, indexerRepo, releaseRepo, app.indexerAPIService)
	app.filterService = filter.NewService(app.log, filterRepo, app.actionService, releaseRepo, app.indexerAPIService, app.indexerService, app.rlsDownloadService, app.notificationService)
	app.releaseService = release.NewService(app.log, app.eventBus, releaseRepo, app.actionService, app.filterService, app.indexerService, app.scheduler)
	app.ircService = irc.NewService(app.log, app.eventBus, app.sse, ircRepo, app.releaseService, app.indexerService, app.proxyService)
	app.feedService = feed.NewService(app.log, app.eventBus, feedRepo, feedCacheRepo, app.releaseService, app.proxyService, app.scheduler)
	app.listService = list.NewService(app.log, app.eventBus, listRepo, app.downloaderService, app.filterService, app.scheduler)
	app.alertService = alert.NewService(app.log, app.eventBus, app.cfg.Config, app.sse, app.updateService, app.ircService, app.listService)

	app.httpServer = http.NewServer(http.Deps{
		Log:                 app.log,
		SSE:                 app.sse,
		DB:                  db,
		Config:              app.cfg,
		SessionManager:      sessionManager,
		Version:             meta.GetVersion(),
		Commit:              meta.GetCommit(),
		Date:                meta.GetDate(),
		ActionService:       app.actionService,
		AlertService:        app.alertService,
		ApiService:          app.apiService,
		AuthService:         app.authService,
		DownloaderService:   app.downloaderService,
		FilterService:       app.filterService,
		FeedService:         app.feedService,
		IndexerService:      app.indexerService,
		IrcService:          app.ircService,
		ListService:         app.listService,
		NotificationService: app.notificationService,
		OIDCService:         oidcService,
		ProxyService:        app.proxyService,
		ReleaseService:      app.releaseService,
		UpdateService:       app.updateService,
	})

	if app.cfg.Config.MetricsEnabled {
		app.metricsManager = metrics.NewMetricsManager(meta.GetVersion(), meta.GetCommit(), meta.GetDate(), app.releaseService, app.ircService, app.feedService, app.listService, app.filterService)
		app.metricsServer = http.NewMetricsServer(app.log, app.cfg, meta.GetVersion(), meta.GetCommit(), meta.GetDate(), app.metricsManager)
	}

	if app.cfg.Config.ProfilingEnabled {
		app.profiler = diagnostics.NewProfiler(app.cfg)
	}

	return nil
}

func (app *App) checkUpdates(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Second):
	}

	app.updateService.CheckUpdates(ctx)
}

func PrintVersion() {
	fmt.Println(meta.GetMetaStr())
}

func PrintVersionJSON() {
	bytes, err := json.Marshal(meta.GetMetaInfo())
	if err != nil {
		return
	}

	fmt.Println(string(bytes))
}
