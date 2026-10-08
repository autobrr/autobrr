// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

type metricsManager interface {
	GetRegistry() *prometheus.Registry
}

type MetricsServer struct {
	log zerolog.Logger

	config *config.AppConfig

	version string
	commit  string
	date    string

	metricsManager metricsManager

	server   *http.Server
	listener net.Listener
}

func NewMetricsServer(log zerolog.Logger, config *config.AppConfig, version string, commit string, date string, metricsManager metricsManager) *MetricsServer {
	return &MetricsServer{
		log:     log.With().Str("module", "http").Logger(),
		config:  config,
		version: version,
		commit:  commit,
		date:    date,

		metricsManager: metricsManager,
	}
}

func (s *MetricsServer) Listen() error {
	listener, err := listen(s.log, fmt.Sprintf("%s:%d", s.config.Config.MetricsHost, s.config.Config.MetricsPort))
	if err != nil {
		return err
	}

	s.listener = listener
	s.server = &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: time.Second * 15,
	}

	s.log.Info().Str("addr", listener.Addr().String()).Msg("starting metrics server")

	return nil
}

func (s *MetricsServer) Serve() error {
	if err := s.server.Serve(s.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *MetricsServer) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	return s.server.Shutdown(ctx)
}

func (s *MetricsServer) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(hlog.NewHandler(s.log))
	r.Use(hlog.RequestIDHandler("request_id", "X-Request-Id"))
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(LoggerMiddleware(&s.log))

	if s.config.Config.MetricsBasicAuthUsers != "" {
		r.Use(BasicAuth("metrics", s.config.Config.MetricsBasicAuthUsers))
	}

	r.Get("/metrics", promhttp.HandlerFor(s.metricsManager.GetRegistry(), promhttp.HandlerOpts{}).ServeHTTP)

	return r
}
