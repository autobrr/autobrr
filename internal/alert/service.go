// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package alert

import (
	"context"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/pkg/version"

	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
)

type updateService interface {
	GetLatestRelease(ctx context.Context) *version.Release
}

type ircService interface {
	UnhealthyNetworks() []domain.IrcUnhealthyNetwork
}

type listService interface {
	List(ctx context.Context, params domain.ListQueryParams) ([]*domain.List, error)
}

type ssePublisher interface {
	Publish(id string, event *sse.Event)
}

type eventBus interface {
	OnAppUpdate(handler func(context.Context, events.AppUpdateEvent) error) func()
	OnIRC(handler func(context.Context, events.IRCEvent) error) func()
	OnListRefresh(handler func(context.Context, events.ListRefreshEvent) error) func()
}

// Service derives the active alerts from the live state of the services that own it.
type Service struct {
	log      zerolog.Logger
	eventBus eventBus
	config   *domain.Config
	sse      ssePublisher

	updateSvc updateService
	ircSvc    ircService
	listSvc   listService
}

func NewService(log zerolog.Logger, eventBus eventBus, config *domain.Config, sse ssePublisher, updateSvc updateService, ircSvc ircService, listSvc listService) *Service {
	s := &Service{
		log:       log.With().Str("module", "alert").Logger(),
		eventBus:  eventBus,
		config:    config,
		sse:       sse,
		updateSvc: updateSvc,
		ircSvc:    ircSvc,
		listSvc:   listSvc,
	}

	s.setupEventListeners()

	return s
}

func (s *Service) setupEventListeners() {
	s.eventBus.OnAppUpdate(func(ctx context.Context, event events.AppUpdateEvent) error {
		s.publishAlertsChanged()
		return nil
	})

	s.eventBus.OnIRC(func(ctx context.Context, event events.IRCEvent) error {
		switch event.Type {
		case events.IRCUnhealthy, events.IRCHealthy:
			s.publishAlertsChanged()
		}

		return nil
	})

	s.eventBus.OnListRefresh(func(ctx context.Context, event events.ListRefreshEvent) error {
		s.publishAlertsChanged()
		return nil
	})
}

// List returns the active alerts. A source that fails is logged and left out, so one broken
// source does not hide the others.
func (s *Service) List(ctx context.Context) []domain.Alert {
	alerts := make([]domain.Alert, 0)

	if s.config.CheckForUpdates {
		if latest := s.updateSvc.GetLatestRelease(ctx); latest != nil {
			alerts = append(alerts, domain.Alert{
				Kind:     domain.AlertKindAppUpdate,
				Severity: domain.AlertSeverityInfo,
				Subject:  latest.TagName,
				URL:      latest.HtmlURL,
				Since:    latest.PublishedAt,
			})
		}
	}

	for _, network := range s.ircSvc.UnhealthyNetworks() {
		alerts = append(alerts, domain.Alert{
			Kind:      domain.AlertKindIRCUnhealthy,
			Severity:  domain.AlertSeverityError,
			SubjectID: network.ID,
			Subject:   network.Name,
			Message:   strings.Join(network.Reasons, "; "),
			Since:     network.Since,
		})
	}

	lists, err := s.listSvc.List(ctx, domain.ListQueryParams{Enabled: new(true), LastRefreshStatus: domain.ListRefreshStatusError})
	if err != nil {
		s.log.Error().Err(err).Msg("could not list failed lists for alerts")
	}

	for _, list := range lists {
		alerts = append(alerts, domain.Alert{
			Kind:      domain.AlertKindListRefreshError,
			Severity:  domain.AlertSeverityError,
			SubjectID: list.ID,
			Subject:   list.Name,
			Message:   list.LastRefreshData,
			Since:     list.LastRefreshTime,
		})
	}

	return alerts
}

// publishAlertsChanged tells connected web clients to refetch the alerts.
func (s *Service) publishAlertsChanged() {
	// r3labs/sse closes the subscriber's stream on an event with empty Data.
	s.sse.Publish(notification.InboxStreamKey, &sse.Event{
		Event: []byte("ALERTS_CHANGED"),
		Data:  []byte("{}"),
	})
}
