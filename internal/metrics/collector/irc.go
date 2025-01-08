// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package collector

import (
	"context"

	"github.com/autobrr/autobrr/internal/irc"
	"github.com/prometheus/client_golang/prometheus"
)

type ircCollector struct {
	ircService irc.Service

	totalCount                    *prometheus.Desc
	enabledCount                  *prometheus.Desc
	connectedCount                *prometheus.Desc
	healthyCount                  *prometheus.Desc
	channelCount                  *prometheus.Desc
	channelEnabledCount           *prometheus.Desc
	channelMonitoringCount        *prometheus.Desc
	channelLastAnnouncedTimestamp *prometheus.Desc
	errorMetric                   *prometheus.Desc
}

func (collector *ircCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.totalCount
	ch <- collector.enabledCount
	ch <- collector.connectedCount
	ch <- collector.channelCount
	ch <- collector.channelEnabledCount
	ch <- collector.channelMonitoringCount
	ch <- collector.channelLastAnnouncedTimestamp
	ch <- collector.errorMetric
}

func (collector *ircCollector) Collect(ch chan<- prometheus.Metric) {
	networks, err := collector.ircService.GetNetworksWithHealth(context.TODO())
	if err != nil {
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}

	enabled := 0
	healthy := 0
	connected := 0
	for _, n := range networks {
		if n.Enabled {
			enabled++
		}
		if n.Connected {
			connected++
		}
		if n.Healthy {
			healthy++
		}

		channelsEnabled := 0
		channelsMonitoring := 0
		for _, c := range n.Channels {
			if c.Enabled {
				channelsEnabled++
			}
			if c.Monitoring {
				channelsMonitoring++
			}
			if !c.LastAnnounce.IsZero() {
				ch <- prometheus.MustNewConstMetric(collector.channelLastAnnouncedTimestamp, prometheus.GaugeValue, float64(int(c.LastAnnounce.Unix())), n.Name, c.Name)
			}
		}
		ch <- prometheus.MustNewConstMetric(collector.channelCount, prometheus.GaugeValue, float64(len(n.Channels)), n.Name)
		ch <- prometheus.MustNewConstMetric(collector.channelEnabledCount, prometheus.GaugeValue, float64(channelsEnabled), n.Name)
		ch <- prometheus.MustNewConstMetric(collector.channelMonitoringCount, prometheus.GaugeValue, float64(channelsMonitoring), n.Name)
	}
	ch <- prometheus.MustNewConstMetric(collector.totalCount, prometheus.GaugeValue, float64(len(networks)))
	ch <- prometheus.MustNewConstMetric(collector.enabledCount, prometheus.GaugeValue, float64(enabled))
	ch <- prometheus.MustNewConstMetric(collector.connectedCount, prometheus.GaugeValue, float64(connected))
	ch <- prometheus.MustNewConstMetric(collector.healthyCount, prometheus.GaugeValue, float64(healthy))
}

func NewIRCCollector(ircService irc.Service) *ircCollector {
	return &ircCollector{
		ircService: ircService,
		totalCount: prometheus.NewDesc(
			"autobrr_irc_total",
			"Number of IRC networks",
			nil,
			nil,
		),
		enabledCount: prometheus.NewDesc(
			"autobrr_irc_enabled_total",
			"Number of enabled IRC networks",
			nil,
			nil,
		),
		connectedCount: prometheus.NewDesc(
			"autobrr_irc_connected_total",
			"Number of connected IRC networks",
			nil,
			nil,
		),
		healthyCount: prometheus.NewDesc(
			"autobrr_irc_healthy_total",
			"Number of healthy IRC networks",
			nil,
			nil,
		),
		channelCount: prometheus.NewDesc(
			"autobrr_irc_channel_total",
			"Number of IRC channel",
			[]string{"network"},
			nil,
		),
		channelEnabledCount: prometheus.NewDesc(
			"autobrr_irc_channel_enabled_total",
			"Number of enabled IRC channel",
			[]string{"network"},
			nil,
		),
		channelMonitoringCount: prometheus.NewDesc(
			"autobrr_irc_channel_monitored_total",
			"Number of IRC channel monitored",
			[]string{"network"},
			nil,
		),
		channelLastAnnouncedTimestamp: prometheus.NewDesc(
			"autobrr_irc_channel_last_announced_timestamp_seconds",
			"The timestamp of the last announced release",
			[]string{"network", "channel"},
			nil,
		),
		errorMetric: prometheus.NewDesc(
			"autobrr_irc_collector_error",
			"Error while collecting irc metrics",
			nil,
			nil,
		),
	}
}
