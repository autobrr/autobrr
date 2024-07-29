// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"bufio"
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
	Settings domain.Notification
	builder  MessageBuilderPlainText

	httpClient *http.Client
}

func NewNtfySender(log zerolog.Logger, settings domain.Notification) domain.NotificationSender {
	return &ntfySender{
		log:      log.With().Str("sender", "ntfy").Logger(),
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
		s.log.Error().Err(err).Msgf("ntfy client request error: %v", event)
		return errors.Wrap(err, "could not create request")
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
		s.log.Error().Err(err).Msgf("ntfy client request error: %v", event)
		return errors.Wrap(err, "could not make request: %+v", req)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(bufio.NewReader(res.Body))
	if err != nil {
		s.log.Error().Err(err).Msgf("ntfy client request error: %v", event)
		return errors.Wrap(err, "could not read data")
	}

	s.log.Trace().Msgf("ntfy status: %v response: %v", res.StatusCode, string(body))

	if res.StatusCode != http.StatusOK {
		s.log.Error().Err(err).Msgf("ntfy client request error: %v", string(body))
		return errors.New("bad status: %v body: %v", res.StatusCode, string(body))
	}

	s.log.Debug().Msg("notification successfully sent to ntfy")

	return nil
}

func (s *ntfySender) CanSend(event domain.NotificationEvent) bool {
	if s.isEnabled() && s.isEnabledEvent(event) {
		return true
	}
	return false
}

func (s *ntfySender) isEnabled() bool {
	if s.Settings.Enabled {
		if s.Settings.Host == "" {
			s.log.Warn().Msg("ntfy missing host")
			return false
		}

		return true
	}

	return false
}

func (s *ntfySender) isEnabledEvent(event domain.NotificationEvent) bool {
	for _, e := range s.Settings.Events {
		if e == string(event) {
			return true
		}
	}

	return false
}
