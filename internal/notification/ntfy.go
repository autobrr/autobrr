// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

type ntfyMessage struct {
	Message string `json:"message"`
	Title   string `json:"title"`
}

type ntfySender struct {
	log      zerolog.Logger
	Settings *domain.Notification
	builder  MessageBuilderPlainText

	httpClient *http.Client
}

func (s *ntfySender) Name() string {
	return "ntfy"
}

func NewNtfySender(log zerolog.Logger, settings *domain.Notification) domain.NotificationSender {
	return &ntfySender{
		log:      log.With().Str("sender", "ntfy").Str("name", settings.Name).Logger(),
		Settings: settings,
		builder:  MessageBuilderPlainText{},
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (s *ntfySender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	m := ntfyMessage{
		Message: s.builder.BuildBody(payload),
		Title:   BuildTitle(event),
	}

	req, err := http.NewRequest(http.MethodPost, s.Settings.Host, strings.NewReader(m.Message))
	if err != nil {
		return errors.Wrap(err, "could not create request for event: %v payload: %v", event, payload)
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", "autobrr")

	req.Header.Set("Title", m.Title)
	if s.Settings.Priority > 0 {
		req.Header.Set("Priority", strconv.Itoa(int(s.Settings.Priority)))
	}

	// set basic auth or access token
	if s.Settings.Username != "" && s.Settings.Password != "" {
		req.SetBasicAuth(s.Settings.Username, s.Settings.Password)
	} else if s.Settings.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.Settings.Token)
	}

	res, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "client request error for event: %v payload: %v", event, payload)
	}

	defer sharedhttp.DrainAndClose(res)

	s.log.Trace().Msgf("ntfy response status: %d", res.StatusCode)

	if res.StatusCode != http.StatusOK {
		// Limit error body reading to prevent memory issues
		limitedReader := io.LimitReader(res.Body, 4096) // 4KB limit
		body, err := io.ReadAll(limitedReader)
		if err != nil {
			return errors.Wrap(err, "could not read body for event: %v payload: %v", event, payload)
		}

		return errors.New("unexpected status: %v body: %v", res.StatusCode, string(body))
	}

	s.log.Debug().Msg("notification successfully sent to ntfy")

	return nil
}

func (s *ntfySender) CanSend(event domain.NotificationEvent) bool {
	if s.IsEnabled() && s.isEnabledEvent(event) {
		return true
	}
	return false
}

func (s *ntfySender) CanSendPayload(event domain.NotificationEvent, payload domain.NotificationPayload) bool {
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

func (s *ntfySender) HasFilterEvents(filterID int) bool {
	if s.Settings.HasFilterNotifications(filterID) {
		return true
	}
	return false
}

func (s *ntfySender) IsEnabled() bool {
	return s.Settings.IsEnabled()
}

func (s *ntfySender) isEnabledEvent(event domain.NotificationEvent) bool {
	return s.Settings.EventEnabled(string(event))
}
