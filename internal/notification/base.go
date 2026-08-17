// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
)

// baseSender holds the settings-driven send decision shared by every sender,
// so the per-(filter, notification) rule lives in exactly one place. Concrete
// senders embed it and keep only transport concerns.
type baseSender struct {
	log      zerolog.Logger
	Settings *domain.Notification
}

func newBaseSender(name string, log zerolog.Logger, settings *domain.Notification) baseSender {
	return baseSender{
		log:      log.With().Str("sender", name).Str("name", settings.Name).Logger(),
		Settings: settings,
	}
}

func (s *baseSender) CanSendPayload(payload domain.NotificationPayload) bool {
	if !s.IsEnabled() {
		return false
	}

	if payload.FilterID > 0 {
		if s.Settings.FilterMuted(payload.FilterID) {
			s.log.Trace().Str("event", string(payload.Event)).Int("filter_id", payload.FilterID).Str("filter", payload.Filter).Msg("notification muted by filter")
			return false
		}

		if s.Settings.FilterEventEnabled(payload.FilterID, payload.Event) {
			return true
		}

		// a per-filter row is authoritative - an event missing from it must not
		// fall back to global events
		if s.Settings.HasFilterNotifications(payload.FilterID) {
			return false
		}

		// scoped to configured filters only - unconfigured filters never fall
		// back to global events, while system events (FilterID == 0) still do
		if s.Settings.FilterScope == domain.NotificationFilterScopeFilterOnly {
			return false
		}
	}

	if s.isEnabledEvent(payload.Event) {
		return true
	}

	return false
}

func (s *baseSender) IsEnabled() bool {
	return s.Settings.IsEnabled()
}

func (s *baseSender) isEnabledEvent(event domain.NotificationEvent) bool {
	return s.Settings.EventEnabled(string(event))
}
