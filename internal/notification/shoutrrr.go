// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"github.com/autobrr/autobrr/internal/domain"

	"github.com/containrrr/shoutrrr"
	"github.com/rs/zerolog"
)

type shoutrrrSender struct {
	log      zerolog.Logger
	Settings *domain.Notification
	builder  MessageBuilderPlainText
}

func (s *shoutrrrSender) Name() string {
	return "shoutrrr"
}

func NewShoutrrrSender(log zerolog.Logger, settings *domain.Notification) domain.NotificationSender {
	return &shoutrrrSender{
		log:      log.With().Str("sender", "shoutrrr").Str("name", settings.Name).Logger(),
		Settings: settings,
		builder:  MessageBuilderPlainText{},
	}
}

func (s *shoutrrrSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	message := s.builder.BuildBody(payload)

	if err := shoutrrr.Send(s.Settings.Host, message); err != nil {
		return err
	}

	s.log.Debug().Msg("notification successfully sent to via shoutrrr")

	return nil
}

func (s *shoutrrrSender) CanSend(event domain.NotificationEvent) bool {
	if s.IsEnabled() && s.isEnabledEvent(event) {
		return true
	}
	return false
}

func (s *shoutrrrSender) CanSendPayload(event domain.NotificationEvent, payload domain.NotificationPayload) bool {
	if !s.IsEnabled() || !s.isEnabledEvent(event) {
		return false
	}

	if payload.FilterID > 0 {
		return s.Settings.FilterEventEnabled(payload.FilterID, event)
	}

	return true
}

func (s *shoutrrrSender) IsEnabled() bool {
	return s.Settings.IsEnabled()
}

func (s *shoutrrrSender) isEnabledEvent(event domain.NotificationEvent) bool {
	return s.Settings.EventEnabled(string(event))
}
