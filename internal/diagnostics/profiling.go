// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package diagnostics

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
)

// Profiler serves the pprof endpoints on their own listener.
type Profiler struct {
	cfg *config.AppConfig

	server   *http.Server
	listener net.Listener
}

func NewProfiler(cfg *config.AppConfig) *Profiler {
	return &Profiler{cfg: cfg}
}

func (p *Profiler) Handler() http.Handler {
	r := chi.NewRouter()

	r.Get("/debug/pprof/", pprof.Index)
	r.Get("/debug/pprof/cmdline", pprof.Cmdline)
	r.Get("/debug/pprof/profile", pprof.Profile)
	r.Get("/debug/pprof/symbol", pprof.Symbol)
	r.Get("/debug/pprof/trace", pprof.Trace)
	r.Get("/debug/pprof/{profile}", pprof.Index)

	return r
}

func (p *Profiler) Listen() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", p.cfg.Config.ProfilingHost, p.cfg.Config.ProfilingPort))
	if err != nil {
		return errors.Wrap(err, "could not listen for profiler")
	}

	p.listener = listener
	p.server = &http.Server{
		Handler:           p.Handler(),
		ReadHeaderTimeout: time.Second * 15,
	}

	return nil
}

func (p *Profiler) Serve() error {
	if err := p.server.Serve(p.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (p *Profiler) Shutdown(ctx context.Context) error {
	if p.server == nil {
		return nil
	}

	return p.server.Shutdown(ctx)
}
