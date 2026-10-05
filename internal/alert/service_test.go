// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package alert

import (
	"context"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/version"

	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubUpdateService struct {
	release *version.Release
}

func (s stubUpdateService) GetLatestRelease(_ context.Context) *version.Release {
	return s.release
}

type stubIrcService struct {
	networks []domain.IrcUnhealthyNetwork
}

func (s stubIrcService) UnhealthyNetworks() []domain.IrcUnhealthyNetwork {
	return s.networks
}

type stubListService struct {
	lists []*domain.List
	err   error
}

// List applies params the way the repo query does, so a wrong filter shows up as a wrong alert.
func (s stubListService) List(_ context.Context, params domain.ListQueryParams) ([]*domain.List, error) {
	if s.err != nil {
		return nil, s.err
	}

	var lists []*domain.List
	for _, list := range s.lists {
		if params.Enabled != nil && list.Enabled != *params.Enabled {
			continue
		}

		if params.LastRefreshStatus != "" && list.LastRefreshStatus != params.LastRefreshStatus {
			continue
		}

		lists = append(lists, list)
	}

	return lists, nil
}

type recordingPublisher struct {
	events []string
}

func (p *recordingPublisher) Publish(_ string, event *sse.Event) {
	p.events = append(p.events, string(event.Event))
}

func TestService_List(t *testing.T) {
	since := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

	release := &version.Release{TagName: "v1.90.0", HtmlURL: "https://github.com/autobrr/autobrr/releases/tag/v1.90.0", PublishedAt: since}
	networks := []domain.IrcUnhealthyNetwork{{ID: 3, Name: "PTP", Reasons: []string{"banned from network: K-Lined", "#ptp-announce: not monitoring"}, Since: since}}
	lists := []*domain.List{
		{ID: 1, Name: "Sonarr", Enabled: true, LastRefreshStatus: domain.ListRefreshStatusError, LastRefreshData: "connection refused", LastRefreshTime: since},
		{ID: 2, Name: "Radarr", Enabled: true, LastRefreshStatus: domain.ListRefreshStatusSuccess},
		{ID: 4, Name: "Trakt", Enabled: false, LastRefreshStatus: domain.ListRefreshStatusError},
	}

	tests := []struct {
		name         string
		checkUpdates bool
		update       stubUpdateService
		irc          stubIrcService
		list         stubListService
		want         []domain.Alert
	}{
		{
			name:         "nothing_active",
			checkUpdates: true,
			want:         []domain.Alert{},
		},
		{
			name:         "every_source",
			checkUpdates: true,
			update:       stubUpdateService{release: release},
			irc:          stubIrcService{networks: networks},
			list:         stubListService{lists: lists},
			want: []domain.Alert{
				{Kind: domain.AlertKindAppUpdate, Severity: domain.AlertSeverityInfo, Subject: "v1.90.0", URL: release.HtmlURL, Since: since},
				{Kind: domain.AlertKindIRCUnhealthy, Severity: domain.AlertSeverityError, SubjectID: 3, Subject: "PTP", Message: "banned from network: K-Lined; #ptp-announce: not monitoring", Since: since},
				{Kind: domain.AlertKindListRefreshError, Severity: domain.AlertSeverityError, SubjectID: 1, Subject: "Sonarr", Message: "connection refused", Since: since},
			},
		},
		{
			name:         "update_check_disabled",
			checkUpdates: false,
			update:       stubUpdateService{release: release},
			want:         []domain.Alert{},
		},
		{
			name:         "failed_source_keeps_others",
			checkUpdates: true,
			irc:          stubIrcService{networks: networks},
			list:         stubListService{err: errors.New("database is locked")},
			want: []domain.Alert{
				{Kind: domain.AlertKindIRCUnhealthy, Severity: domain.AlertSeverityError, SubjectID: 3, Subject: "PTP", Message: "banned from network: K-Lined; #ptp-announce: not monitoring", Since: since},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &domain.Config{CheckForUpdates: tt.checkUpdates}
			s := NewService(zerolog.Nop(), events.NewEventBus(zerolog.Nop()), config, &recordingPublisher{}, tt.update, tt.irc, tt.list)

			assert.Equal(t, tt.want, s.List(t.Context()))
		})
	}
}

func TestService_PublishesAlertsChanged(t *testing.T) {
	bus := events.NewEventBus(zerolog.Nop())
	publisher := &recordingPublisher{}
	NewService(zerolog.Nop(), bus, &domain.Config{}, publisher, stubUpdateService{}, stubIrcService{}, stubListService{})

	bus.EmitAppUpdate(t.Context(), events.AppUpdateEvent{Type: events.ApplicationUpdate})
	bus.EmitIRC(t.Context(), events.IRCEvent{Type: events.IRCDisconnected})
	bus.EmitIRC(t.Context(), events.IRCEvent{Type: events.IRCUnhealthy})
	bus.EmitIRC(t.Context(), events.IRCEvent{Type: events.IRCHealthy})
	bus.EmitListRefresh(t.Context(), events.ListRefreshEvent{Type: events.ListRefreshError})

	require.Len(t, publisher.events, 4, "IRC events other than unhealthy and healthy do not change the alerts")
	for _, event := range publisher.events {
		assert.Equal(t, "ALERTS_CHANGED", event)
	}
}
