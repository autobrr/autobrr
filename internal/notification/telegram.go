// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

// TelegramMessage Reference: https://core.telegram.org/bots/api#sendmessage
type TelegramMessage struct {
	ChatID          string `json:"chat_id"`
	Text            string `json:"text"`
	ParseMode       string `json:"parse_mode"`
	MessageThreadID int    `json:"message_thread_id,omitempty"`
}

type telegramSender struct {
	log      zerolog.Logger
	Settings *domain.Notification
	ThreadID int
	builder  MessageBuilderHTML

	httpClient *http.Client
}

func (s *telegramSender) Name() string {
	return "telegram"
}

func NewTelegramSender(log zerolog.Logger, settings *domain.Notification) domain.NotificationSender {
	threadID := 0
	if t := settings.Topic; t != "" {
		var err error
		threadID, err = strconv.Atoi(t)
		if err != nil {
			log.Error().Err(err).Msgf("could not parse specified topic %q as an integer", t)
		}
	}
	return &telegramSender{
		log:      log.With().Str("sender", "telegram").Str("name", settings.Name).Logger(),
		Settings: settings,
		ThreadID: threadID,
		builder:  MessageBuilderHTML{},
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (s *telegramSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	payload.Sender = s.Settings.Username

	message := s.builder.BuildBody(payload)
	m := TelegramMessage{
		ChatID:          s.Settings.Channel,
		Text:            message,
		MessageThreadID: s.ThreadID,
		ParseMode:       "HTML",
		//ParseMode: "MarkdownV2",
	}

	jsonData, err := json.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "could not marshal json request for event: %v payload: %v", event, payload)
	}

	var host string

	if s.Settings.Host == "" {
		host = "https://api.telegram.org"
	} else {
		host = s.Settings.Host
	}

	url := fmt.Sprintf("%v/bot%v/sendMessage", host, s.Settings.Token)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.Wrap(err, "could not create request for event: %v payload: %v", event, payload)
	}

	req.Header.Set("Content-Type", "application/json")
	//req.Header.Set("User-Agent", "autobrr")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "client request error for event: %v payload: %v", event, payload)
	}

	defer res.Body.Close()

	s.log.Trace().Msgf("telegram status: %d", res.StatusCode)

	if res.StatusCode != http.StatusOK {
		body, err := io.ReadAll(bufio.NewReader(res.Body))
		if err != nil {
			return errors.Wrap(err, "could not read body for event: %v payload: %v", event, payload)
		}

		return errors.New("unexpected status: %v body: %v", res.StatusCode, string(body))
	}

	s.log.Debug().Msg("notification successfully sent to telegram")

	return nil
}

func (s *telegramSender) CanSend(event domain.NotificationEvent) bool {
	if s.IsEnabled() && s.isEnabledEvent(event) {
		return true
	}
	return false
}

func (s *telegramSender) CanSendPayload(event domain.NotificationEvent, payload domain.NotificationPayload) bool {
	if !s.IsEnabled() {
		return false
	}

	if s.Settings.FilterMuted(payload.FilterID) {
		s.log.Trace().Str("event", string(event)).Int("filter_id", payload.FilterID).Str("filter", payload.Filter).Msg("notification muted by filter")
		return false
	}

	if s.Settings.FilterEventEnabled(payload.FilterID, event) {
		return true
	}

	if s.isEnabledEvent(event) {
		return true
	}

	return false
}

func (s *telegramSender) IsEnabled() bool {
	return s.Settings.IsEnabled()
}

func (s *telegramSender) isEnabledEvent(event domain.NotificationEvent) bool {
	return s.Settings.EventEnabled(string(event))
}
