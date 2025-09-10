// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

type pushoverMessage struct {
	Token     string    `json:"api_key"`
	User      string    `json:"token"`
	Message   string    `json:"message"`
	Priority  int32     `json:"priority"`
	Title     string    `json:"title"`
	Timestamp time.Time `json:"timestamp"`
	Html      int       `json:"html,omitempty"`
}

type pushoverSender struct {
	log      zerolog.Logger
	Settings *domain.Notification
	baseUrl  string
	builder  MessageBuilderHTML

	httpClient *http.Client
}

func (s *pushoverSender) Name() string {
	return "pushover"
}

func NewPushoverSender(log zerolog.Logger, settings *domain.Notification) domain.NotificationSender {
	return &pushoverSender{
		log:      log.With().Str("sender", "pushover").Str("name", settings.Name).Logger(),
		Settings: settings,
		baseUrl:  "https://api.pushover.net/1/messages.json",
		builder:  MessageBuilderHTML{},
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (s *pushoverSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	title := BuildTitle(event)
	message := s.builder.BuildBody(payload)

	m := pushoverMessage{
		Token:     s.Settings.APIKey,
		User:      s.Settings.Token,
		Priority:  s.Settings.Priority,
		Message:   message,
		Title:     title,
		Timestamp: time.Now(),
		Html:      1,
	}

	data := url.Values{}
	data.Set("token", m.Token)
	data.Set("user", m.User)
	data.Set("message", m.Message)
	data.Set("priority", strconv.Itoa(int(m.Priority)))
	data.Set("title", m.Title)
	data.Set("timestamp", fmt.Sprintf("%v", m.Timestamp.Unix()))
	data.Set("html", fmt.Sprintf("%v", m.Html))

	if m.Priority == 2 {
		data.Set("expire", "3600")
		data.Set("retry", "60")
	}

	req, err := http.NewRequest(http.MethodPost, s.baseUrl, strings.NewReader(data.Encode()))
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

	s.log.Trace().Msgf("pushover response status: %d", res.StatusCode)

	if res.StatusCode != http.StatusOK {
		body, err := io.ReadAll(bufio.NewReader(res.Body))
		if err != nil {
			return errors.Wrap(err, "could not read body for event: %v payload: %v", event, payload)
		}

		return errors.New("unexpected status: %v body: %v", res.StatusCode, string(body))
	}

	s.log.Debug().Msg("notification successfully sent to pushover")

	return nil
}

func (s *pushoverSender) CanSend(event domain.NotificationEvent) bool {
	if s.IsEnabled() && s.isEnabledEvent(event) {
		return true
	}
	return false
}

func (s *pushoverSender) CanSendPayload(event domain.NotificationEvent, payload domain.NotificationPayload) bool {
	if !s.IsEnabled() || !s.isEnabledEvent(event) {
		return false
	}

	if payload.FilterID > 0 {
		return s.Settings.FilterEventEnabled(payload.FilterID, event)
	}

	return true
}

func (s *pushoverSender) IsEnabled() bool {
	return s.Settings.IsEnabled()
}

func (s *pushoverSender) isEnabledEvent(event domain.NotificationEvent) bool {
	return s.Settings.EventEnabled(string(event))
}
