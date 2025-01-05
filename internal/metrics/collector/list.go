// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package collector

import (
	"context"

	"github.com/autobrr/autobrr/internal/list"
	"github.com/prometheus/client_golang/prometheus"
)

type listCollector struct {
	listService list.Service

	totalCount           *prometheus.Desc
	enabledCount         *prometheus.Desc
	LastRefreshTimestamp *prometheus.Desc
	errorMetric          *prometheus.Desc
}

func (collector *listCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.totalCount
	ch <- collector.enabledCount
	ch <- collector.LastRefreshTimestamp
	ch <- collector.errorMetric
}

func (collector *listCollector) Collect(ch chan<- prometheus.Metric) {
	lists, err := collector.listService.List(context.TODO())
	if err != nil {
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}

	enabled := 0
	for _, l := range lists {
		if l.Enabled {
			enabled++
		}

		if !l.LastRefreshTime.IsZero() {
			ch <- prometheus.MustNewConstMetric(collector.LastRefreshTimestamp, prometheus.GaugeValue, float64(int(l.LastRefreshTime.Unix())), l.Name)
		}
	}
	ch <- prometheus.MustNewConstMetric(collector.totalCount, prometheus.GaugeValue, float64(len(lists)))
	ch <- prometheus.MustNewConstMetric(collector.enabledCount, prometheus.GaugeValue, float64(enabled))
}

func NewListCollector(listService list.Service) *listCollector {
	return &listCollector{
		listService: listService,
		totalCount: prometheus.NewDesc(
			"autobrr_list_total",
			"Number of lists",
			nil,
			nil,
		),
		enabledCount: prometheus.NewDesc(
			"autobrr_list_enabled_total",
			"Number of enabled lists",
			nil,
			nil,
		),
		LastRefreshTimestamp: prometheus.NewDesc(
			"autobrr_list_last_refresh_timestamp_seconds",
			"The timestamp of the last list run",
			[]string{"list"},
			nil,
		),
		errorMetric: prometheus.NewDesc(
			"autobrr_list_collector_error",
			"Error while collecting list metrics",
			nil,
			nil,
		),
	}
}
