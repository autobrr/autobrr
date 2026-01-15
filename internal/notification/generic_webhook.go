// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

type genericWebhookSender struct {
	log      zerolog.Logger
	Settings *domain.Notification

	httpClient *http.Client
}

func NewGenericWebhookSender(log zerolog.Logger, settings *domain.Notification) domain.NotificationSender {
	return &genericWebhookSender{
		log:      log.With().Str("sender", "generic_webhook").Str("name", settings.Name).Logger(),
		Settings: settings,
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (s *genericWebhookSender) Name() string {
	return "generic_webhook"
}

func (s *genericWebhookSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	// Build the full payload with all release data
	webhookPayload := domain.NewGenericWebhookPayload(payload, payload.Release)

	jsonData, err := json.Marshal(webhookPayload)
	if err != nil {
		return errors.Wrap(err, "could not marshal json request for event: %v", event)
	}

	req, err := http.NewRequest(http.MethodPost, s.Settings.Webhook, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.Wrap(err, "could not create request for event: %v", event)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not make request for event: %v", event)
	}

	defer sharedhttp.DrainAndClose(res)

	s.log.Trace().Msgf("generic webhook response status: %d", res.StatusCode)

	// Accept 2xx status codes as success
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		// Limit error body reading to prevent memory issues
		limitedReader := io.LimitReader(res.Body, 4096) // 4KB limit
		body, err := io.ReadAll(limitedReader)
		if err != nil {
			return errors.Wrap(err, "could not read body for event: %v", event)
		}

		return errors.New("unexpected status: %v body: %v", res.StatusCode, string(body))
	}

	s.log.Debug().Str("event", string(event)).Msg("notification successfully sent to generic webhook")

	return nil
}

func (s *genericWebhookSender) CanSend(event domain.NotificationEvent) bool {
	if s.IsEnabled() && s.isEnabledEvent(event) {
		return true
	}
	return false
}

func (s *genericWebhookSender) CanSendPayload(event domain.NotificationEvent, payload domain.NotificationPayload) bool {
	if !s.IsEnabled() {
		return false
	}

	if payload.FilterID > 0 {
		if s.Settings.FilterMuted(payload.FilterID) {
			s.log.Trace().Str("event", string(event)).Int("filter_id", payload.FilterID).Str("filter", payload.Filter).Msg("notification muted by filter")
			return false
		}

		// Check if the filter has custom notifications configured
		if s.Settings.FilterEventEnabled(payload.FilterID, event) {
			return true
		}

		// If the filter has custom notifications but the event is not enabled, don't fall back to global
		if s.Settings.HasFilterNotifications(payload.FilterID) {
			return false
		}
	}

	// Fall back to global events for non-filter events or filters without custom notifications
	if s.isEnabledEvent(event) {
		return true
	}

	return false
}

func (s *genericWebhookSender) HasFilterEvents(filterID int) bool {
	if s.Settings.HasFilterNotifications(filterID) {
		return true
	}
	return false
}

func (s *genericWebhookSender) IsEnabled() bool {
	return s.Settings.IsEnabled()
}

func (s *genericWebhookSender) isEnabledEvent(event domain.NotificationEvent) bool {
	return s.Settings.EventEnabled(string(event))
}
