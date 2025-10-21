// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

// unsure if this is the best approach to send an image with the notification
const defaultImageURL = "https://raw.githubusercontent.com/autobrr/autobrr/master/.github/images/logo.png"

type LunaSeaMessage struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Image string `json:"image,omitempty"`
}

type lunaSeaSender struct {
	log      zerolog.Logger
	Settings *domain.Notification
	builder  MessageBuilderPlainText

	httpClient *http.Client
}

func (s *lunaSeaSender) Name() string {
	return "lunasea"
}

var lunaWebhook = regexp.MustCompile(`/(radarr|sonarr|lidarr|tautulli|overseerr)/`)

func (s *lunaSeaSender) rewriteWebhookURL(url string) string {
	return lunaWebhook.ReplaceAllString(url, "/custom/")
} // `custom` is not mentioned in their docs, so I thought this would be a good idea to add to avoid user errors

func NewLunaSeaSender(log zerolog.Logger, settings *domain.Notification) domain.NotificationSender {
	return &lunaSeaSender{
		log:      log.With().Str("sender", "lunasea").Str("name", settings.Name).Logger(),
		Settings: settings,
		builder:  MessageBuilderPlainText{},
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (s *lunaSeaSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	m := LunaSeaMessage{
		Title: BuildTitle(event),
		Body:  s.builder.BuildBody(payload),
		Image: defaultImageURL,
	}

	jsonData, err := json.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "could not marshal json request for event: %v payload: %v", event, payload)
	}

	rewrittenURL := s.rewriteWebhookURL(s.Settings.Webhook)

	req, err := http.NewRequest(http.MethodPost, rewrittenURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.Wrap(err, "could not create request for event: %v payload: %v", event, payload)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "client request error for event: %v payload: %v", event, payload)
	}

	defer sharedhttp.DrainAndClose(res)

	if res.StatusCode != http.StatusOK {
		// Limit error body reading to prevent memory issues
		limitedReader := io.LimitReader(res.Body, 4096) // 4KB limit
		body, err := io.ReadAll(limitedReader)
		if err != nil {
			return errors.Wrap(err, "could not read body for event: %v payload: %v", event, payload)
		}

		return errors.New("unexpected status: %v body: %v", res.StatusCode, string(body))
	}

	s.log.Debug().Msg("notification successfully sent to lunasea")

	return nil
}

func (s *lunaSeaSender) CanSend(event domain.NotificationEvent) bool {
	return s.IsEnabled() && s.isEnabledEvent(event)
}

func (s *lunaSeaSender) CanSendPayload(event domain.NotificationEvent, payload domain.NotificationPayload) bool {
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

func (s *lunaSeaSender) HasFilterEvents(filterID int) bool {
	if s.Settings.HasFilterNotifications(filterID) {
		return true
	}
	return false
}

func (s *lunaSeaSender) IsEnabled() bool {
	return s.Settings.IsEnabled()
}

func (s *lunaSeaSender) isEnabledEvent(event domain.NotificationEvent) bool {
	return s.Settings.EventEnabled(string(event))
}
