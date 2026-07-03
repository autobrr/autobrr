// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package metrics

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/metrics/collector"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type ircService interface {
	GetNetworksWithHealth(ctx context.Context) ([]domain.IrcNetworkWithHealth, error)
}

type filterService interface {
	ListFilters(ctx context.Context) ([]domain.Filter, error)
}

type feedService interface {
	Find(ctx context.Context) ([]domain.Feed, error)
}

type listService interface {
	List(ctx context.Context) ([]*domain.List, error)
}

type releaseService interface {
	Stats(ctx context.Context) (*domain.ReleaseStats, error)
}

type Manager struct {
	registry *prometheus.Registry
}

func NewMetricsManager(version string, commit string, date string, releaseService releaseService, ircService ircService, feedService feedService, listService listService, filterService filterService) *Manager {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name: "autobrr_info",
				Help: "Autobrr version information",
				ConstLabels: prometheus.Labels{
					"version":    version,
					"build_time": date,
					"revision":   commit,
				},
			},
			func() float64 { return 1 },
		),
		collector.NewReleaseCollector(releaseService),
		collector.NewIRCCollector(ircService),
		collector.NewFeedCollector(feedService),
		collector.NewListCollector(listService),
		collector.NewFilterCollector(filterService),
	)
	return &Manager{
		registry: registry,
	}
}

func (s *Manager) GetRegistry() *prometheus.Registry {
	return s.registry
}
