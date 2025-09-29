// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package collector

import (
	"context"

	"github.com/autobrr/autobrr/internal/release"
	"github.com/prometheus/client_golang/prometheus"
)

type releaseCollector struct {
	releaseService release.Service

	totalCount          *prometheus.Desc
	filteredCount       *prometheus.Desc
	filterRejectedCount *prometheus.Desc
	pushApprovedCount   *prometheus.Desc
	pushRejectedCount   *prometheus.Desc
	pushErrorCount      *prometheus.Desc
	errorMetric         *prometheus.Desc
}

func (collector *releaseCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.totalCount
	ch <- collector.filteredCount
	ch <- collector.filterRejectedCount
	ch <- collector.pushApprovedCount
	ch <- collector.pushRejectedCount
	ch <- collector.pushErrorCount
	ch <- collector.errorMetric
}

func (collector *releaseCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := collector.releaseService.Stats(context.TODO())
	if err != nil {
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(collector.totalCount, prometheus.GaugeValue, float64(stats.TotalCount))
	ch <- prometheus.MustNewConstMetric(collector.filteredCount, prometheus.GaugeValue, float64(stats.FilteredCount))
	ch <- prometheus.MustNewConstMetric(collector.filterRejectedCount, prometheus.GaugeValue, float64(stats.FilterRejectedCount))
	ch <- prometheus.MustNewConstMetric(collector.pushApprovedCount, prometheus.GaugeValue, float64(stats.PushApprovedCount))
	ch <- prometheus.MustNewConstMetric(collector.pushRejectedCount, prometheus.GaugeValue, float64(stats.PushRejectedCount))
	ch <- prometheus.MustNewConstMetric(collector.pushErrorCount, prometheus.GaugeValue, float64(stats.PushErrorCount))
}

func NewReleaseCollector(releaseService release.Service) *releaseCollector {
	return &releaseCollector{
		releaseService: releaseService,
		totalCount: prometheus.NewDesc(
			"autobrr_release_total",
			"Number of releases",
			nil,
			nil,
		),
		filteredCount: prometheus.NewDesc(
			"autobrr_release_filtered_total",
			"Number of releases filtered",
			nil,
			nil,
		),
		filterRejectedCount: prometheus.NewDesc(
			"autobrr_release_filter_rejected_total",
			"Number of releases that got rejected because of a filter",
			nil,
			nil,
		),
		pushApprovedCount: prometheus.NewDesc(
			"autobrr_release_push_approved_total",
			"Number of releases push approved",
			nil,
			nil,
		),
		pushRejectedCount: prometheus.NewDesc(
			"autobrr_release_push_rejected_total",
			"Number of releases push rejected",
			nil,
			nil,
		),
		pushErrorCount: prometheus.NewDesc(
			"autobrr_release_push_error_total",
			"Number of releases push errored",
			nil,
			nil,
		),
		errorMetric: prometheus.NewDesc(
			"autobrr_release_collector_error",
			"Error while collecting release metrics",
			nil,
			nil,
		),
	}
}
