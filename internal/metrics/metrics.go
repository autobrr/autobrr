// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package metrics

import (
	"github.com/autobrr/autobrr/internal/feed"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/irc"
	"github.com/autobrr/autobrr/internal/list"
	"github.com/autobrr/autobrr/internal/metrics/collector"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/prometheus/client_golang/prometheus"
)

type MetricsManager struct {
	registry *prometheus.Registry
}

func NewMetricsManager(version string, commit string, date string, releaseService release.Service, ircService irc.Service, feedService feed.Service, listService list.Service, filterService filter.Service) *MetricsManager {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
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
	return &MetricsManager{
		registry: registry,
	}
}

func (s *MetricsManager) GetRegistry() *prometheus.Registry {
	return s.registry
}
