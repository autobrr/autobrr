// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package collector

import (
	"context"

	"github.com/autobrr/autobrr/internal/feed"
	"github.com/prometheus/client_golang/prometheus"
)

type feedCollector struct {
	feedService feed.Service

	totalCount       *prometheus.Desc
	enabledCount     *prometheus.Desc
	LastRunTimestamp *prometheus.Desc
	NextRunTimestamp *prometheus.Desc
	errorMetric      *prometheus.Desc
}

func (collector *feedCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.totalCount
	ch <- collector.enabledCount
	ch <- collector.LastRunTimestamp
	ch <- collector.NextRunTimestamp
	ch <- collector.errorMetric
}

func (collector *feedCollector) Collect(ch chan<- prometheus.Metric) {
	feeds, err := collector.feedService.Find(context.TODO())
	if err != nil {
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}

	enabled := 0
	for _, f := range feeds {
		if f.Enabled {
			enabled++
		}

		if !f.LastRun.IsZero() {
			ch <- prometheus.MustNewConstMetric(collector.LastRunTimestamp, prometheus.GaugeValue, float64(int(f.LastRun.Unix())), f.Name)
		}
		if !f.NextRun.IsZero() {
			ch <- prometheus.MustNewConstMetric(collector.NextRunTimestamp, prometheus.GaugeValue, float64(int(f.NextRun.Unix())), f.Name)
		}
	}
	ch <- prometheus.MustNewConstMetric(collector.totalCount, prometheus.GaugeValue, float64(len(feeds)))
	ch <- prometheus.MustNewConstMetric(collector.enabledCount, prometheus.GaugeValue, float64(enabled))
}

func NewFeedCollector(feedService feed.Service) *feedCollector {
	return &feedCollector{
		feedService: feedService,
		totalCount: prometheus.NewDesc(
			"autobrr_feed_total",
			"Number of feeds",
			nil,
			nil,
		),
		enabledCount: prometheus.NewDesc(
			"autobrr_feed_enabled_total",
			"Number of enabled feeds",
			nil,
			nil,
		),
		LastRunTimestamp: prometheus.NewDesc(
			"autobrr_feed_last_run_timestamp_seconds",
			"The timestamp of the last feed run",
			[]string{"feed"},
			nil,
		),
		NextRunTimestamp: prometheus.NewDesc(
			"autobrr_feed_next_run_timestamp_seconds",
			"The timestamp of the next feed run",
			[]string{"feed"},
			nil,
		),
		errorMetric: prometheus.NewDesc(
			"autobrr_feed_collector_error",
			"Error while collecting feed metrics",
			nil,
			nil,
		),
	}
}
