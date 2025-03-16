// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

type metricsManager interface {
	GetRegistry() *prometheus.Registry
}

type MetricsServer struct {
	log zerolog.Logger

	metricsManager metricsManager

	config *config.AppConfig

	version string
	commit  string
	date    string
}

func NewMetricsServer(log logger.Logger, config *config.AppConfig, version string, commit string, date string, metricsManager metricsManager) MetricsServer {
	return MetricsServer{
		log:     log.With().Str("module", "http").Logger(),
		config:  config,
		version: version,
		commit:  commit,
		date:    date,

		metricsManager: metricsManager,
	}
}

func (s MetricsServer) Open() error {
	addr := fmt.Sprintf("%v:%v", s.config.Config.MetricsHost, s.config.Config.MetricsPort)

	var err error
	for _, proto := range []string{"tcp", "tcp4", "tcp6"} {
		if err = s.tryToServe(addr, proto); err == nil {
			break
		}

		s.log.Error().Err(err).Msgf("Failed to start %s server. Attempted to listen on %s", proto, addr)
	}

	return err
}

func (s MetricsServer) tryToServe(addr, protocol string) error {
	listener, err := net.Listen(protocol, addr)
	if err != nil {
		return err
	}

	s.log.Info().Msgf("Starting Metrics %s server. Listening on %s", protocol, listener.Addr().String())

	server := http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: time.Second * 15,
	}

	return server.Serve(listener)
}

func (s MetricsServer) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(LoggerMiddleware(&s.log))

	if s.config.Config.MetricsBasicAuthUsers != "" {
		r.Use(BasicAuth("metrics", s.config.Config.MetricsBasicAuthUsers))
	}

	r.Get("/metrics", promhttp.HandlerFor(s.metricsManager.GetRegistry(), promhttp.HandlerOpts{}).ServeHTTP)

	return r
}
