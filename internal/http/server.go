// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
	"github.com/r3labs/sse/v2"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
)

type Server struct {
	log zerolog.Logger
	sse *sse.Server
	db  *database.DB

	config      *config.AppConfig
	cookieStore *sessions.CookieStore

	version string
	commit  string
	date    string

	actionService         actionService
	apiService            apikeyService
	authService           authService
	downloadClientService downloadClientService
	filterService         filterService
	feedService           feedService
	indexerService        indexerService
	ircService            ircService
	notificationService   notificationService
	proxyService          proxyService
	releaseService        releaseService
	updateService         updateService

	logger logger.Logger
}

func NewServer(log logger.Logger, config *config.AppConfig, sse *sse.Server, db *database.DB, version string, commit string, date string, actionService actionService, apiService apikeyService, authService authService, downloadClientSvc downloadClientService, filterSvc filterService, feedSvc feedService, indexerSvc indexerService, ircSvc ircService, notificationSvc notificationService, proxySvc proxyService, releaseSvc releaseService, updateSvc updateService) Server {
	return Server{
		log:     log.With().Str("module", "http").Logger(),
		logger:  log,
		config:  config,
		sse:     sse,
		db:      db,
		version: version,
		commit:  commit,
		date:    date,

		cookieStore: sessions.NewCookieStore([]byte(config.Config.SessionSecret)),

		actionService:         actionService,
		apiService:            apiService,
		authService:           authService,
		downloadClientService: downloadClientSvc,
		filterService:         filterSvc,
		feedService:           feedSvc,
		indexerService:        indexerSvc,
		ircService:            ircSvc,
		notificationService:   notificationSvc,
		proxyService:          proxySvc,
		releaseService:        releaseSvc,
		updateService:         updateSvc,
	}
}

func (s Server) Open() error {
	addr := fmt.Sprintf("%v:%v", s.config.Config.Host, s.config.Config.Port)

	var err error
	for _, proto := range []string{"tcp", "tcp4", "tcp6"} {
		if err = s.tryToServe(addr, proto); err == nil {
			break
		}

		s.log.Error().Err(err).Msgf("Failed to start %s server. Attempted to listen on %s", proto, addr)
	}

	return err
}

func (s Server) tryToServe(addr, protocol string) error {
	listener, err := net.Listen(protocol, addr)
	if err != nil {
		return err
	}

	s.log.Info().Msgf("Starting API %s server. Listening on %s", protocol, listener.Addr().String())

	server := http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: time.Second * 15,
	}

	return server.Serve(listener)
}

func (s Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(LoggerMiddleware(&s.log))

	c := cors.New(cors.Options{
		AllowCredentials:   true,
		AllowedMethods:     []string{"HEAD", "OPTIONS", "GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowOriginFunc:    func(origin string) bool { return true },
		OptionsPassthrough: true,
		// Enable Debugging for testing, consider disabling in production
		Debug: false,
	})

	r.Use(c.Handler)

	encoder := newEncoder(s.logger)

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", newAuthHandler(encoder, s.log, s, s.config.Config, s.cookieStore, s.authService).Routes)
		r.Route("/healthz", newHealthHandler(encoder, s.db).Routes)

		r.Group(func(r chi.Router) {
			r.Use(s.IsAuthenticated)

			r.Route("/actions", newActionHandler(encoder, s.actionService).Routes)
			r.Route("/config", newConfigHandler(encoder, s, s.config).Routes)
			r.Route("/download_clients", newDownloadClientHandler(encoder, s.downloadClientService).Routes)
			r.Route("/filters", newFilterHandler(encoder, s.filterService).Routes)
			r.Route("/feeds", newFeedHandler(encoder, s.feedService).Routes)
			r.Route("/irc", newIrcHandler(encoder, s.sse, s.ircService).Routes)
			r.Route("/indexer", newIndexerHandler(encoder, s.indexerService, s.ircService).Routes)
			r.Route("/keys", newAPIKeyHandler(encoder, s.apiService).Routes)
			r.Route("/logs", newLogsHandler(s.config).Routes)
			r.Route("/notification", newNotificationHandler(encoder, s.notificationService).Routes)
			r.Route("/proxy", newProxyHandler(encoder, s.proxyService).Routes)
			r.Route("/release", newReleaseHandler(encoder, s.releaseService).Routes)
			r.Route("/updates", newUpdateHandler(encoder, s.updateService).Routes)

			r.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {

				// inject CORS headers to bypass checks
				s.sse.Headers = map[string]string{
					"Content-Type":      "text/event-stream",
					"Cache-Control":     "no-cache",
					"Connection":        "keep-alive",
					"X-Accel-Buffering": "no",
				}

				s.sse.ServeHTTP(w, r)
			})
		})
	})

	// serve the web
	web.RegisterHandler(r, s.version, s.config.Config.BaseURL)

	return r
}

func (s Server) index(w http.ResponseWriter, r *http.Request) {
	p := web.IndexParams{
		Title:   "Dashboard",
		Version: s.version,
		BaseUrl: s.config.Config.BaseURL,
	}
	web.Index(w, p)
}
