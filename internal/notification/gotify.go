// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

type gotifyMessage struct {
	Message string `json:"message"`
	Title   string `json:"title"`
}

type gotifySender struct {
	log      zerolog.Logger
	Settings domain.Notification
	builder  MessageBuilderPlainText

	httpClient *http.Client
}

func (s *gotifySender) Name() string {
	return "gotify"
}

func NewGotifySender(log zerolog.Logger, settings domain.Notification) domain.NotificationSender {
	return &gotifySender{
		log:      log.With().Str("sender", "gotify").Logger(),
		Settings: settings,
		builder:  MessageBuilderPlainText{},
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (s *gotifySender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	m := gotifyMessage{
		Message: s.builder.BuildBody(payload),
		Title:   BuildTitle(event),
	}

	data := url.Values{}
	data.Set("message", m.Message)
	data.Set("title", m.Title)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%v/message?token=%v", s.Settings.Host, s.Settings.Token), strings.NewReader(data.Encode()))
	if err != nil {
		return errors.Wrap(err, "could not create request for event: %v payload: %v", event, payload)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "autobrr")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "client request error for event: %v payload: %v", event, payload)
	}

	defer res.Body.Close()

	s.log.Trace().Msgf("gotify status: %d", res.StatusCode)

	if res.StatusCode != http.StatusOK {
		body, err := io.ReadAll(bufio.NewReader(res.Body))
		if err != nil {
			return errors.Wrap(err, "could not read body for event: %v payload: %v", event, payload)
		}

		return errors.New("unexpected status: %v body: %v", res.StatusCode, string(body))
	}

	s.log.Debug().Msg("notification successfully sent to gotify")

	return nil
}

func (s *gotifySender) CanSend(event domain.NotificationEvent) bool {
	if s.isEnabled() && s.isEnabledEvent(event) {
		return true
	}
	return false
}

func (s *gotifySender) isEnabled() bool {
	if s.Settings.Enabled {
		if s.Settings.Host == "" {
			s.log.Warn().Msg("gotify missing host")
			return false
		}

		if s.Settings.Token == "" {
			s.log.Warn().Msg("gotify missing application token")
			return false
		}

		return true
	}

	return false
}

func (s *gotifySender) isEnabledEvent(event domain.NotificationEvent) bool {
	for _, e := range s.Settings.Events {
		if e == string(event) {
			return true
		}
	}

	return false
}
