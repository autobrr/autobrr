// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package collector

import (
	"context"

	"github.com/autobrr/autobrr/internal/filter"
	"github.com/prometheus/client_golang/prometheus"
)

type filterCollector struct {
	filterService filter.Service

	totalCount   *prometheus.Desc
	enabledCount *prometheus.Desc
	errorMetric  *prometheus.Desc
}

func (collector *filterCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.totalCount
	ch <- collector.enabledCount
	ch <- collector.errorMetric
}

func (collector *filterCollector) Collect(ch chan<- prometheus.Metric) {
	lists, err := collector.filterService.ListFilters(context.TODO())
	if err != nil {
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}

	enabled := 0
	for _, f := range lists {
		if f.Enabled {
			enabled++
		}
	}
	ch <- prometheus.MustNewConstMetric(collector.totalCount, prometheus.GaugeValue, float64(len(lists)))
	ch <- prometheus.MustNewConstMetric(collector.enabledCount, prometheus.GaugeValue, float64(enabled))
}

func NewFilterCollector(filterService filter.Service) *filterCollector {
	return &filterCollector{
		filterService: filterService,
		totalCount: prometheus.NewDesc(
			"autobrr_filter_total",
			"Number of filters",
			nil,
			nil,
		),
		enabledCount: prometheus.NewDesc(
			"autobrr_filter_enabled_total",
			"Number of enabled filters",
			nil,
			nil,
		),
		errorMetric: prometheus.NewDesc(
			"autobrr_filter_collector_error",
			"Error while collecting filter metrics",
			nil,
			nil,
		),
	}
}
