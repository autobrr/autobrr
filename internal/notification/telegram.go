// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
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

func NewTelegramSender(log zerolog.Logger, settings *domain.Notification) Sender {
	threadID := 0
	if t := settings.Topic; t != "" {
		var err error
		threadID, err = strconv.Atoi(t)
		if err != nil {
			log.Error().Err(err).Str("topic", t).Msg("could not parse specified topic as an integer")
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

	url := fmt.Sprintf("%s/bot%s/sendMessage", host, s.Settings.Token)

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

	defer sharedhttp.DrainAndClose(res)

	s.log.Trace().Int("status_code", res.StatusCode).Msg("response status")

	if res.StatusCode != http.StatusOK {
		// Limit error body reading to prevent memory issues
		limitedReader := io.LimitReader(res.Body, 4096) // 4KB limit
		body, err := io.ReadAll(limitedReader)
		if err != nil {
			return errors.Wrap(err, "could not read body for event: %v payload: %v", event, payload)
		}

		return errors.New("unexpected status: %v body: %v", res.StatusCode, string(body))
	}

	s.log.Debug().Msg("notification successfully sent to telegram")

	return nil
}

func (s *telegramSender) IsEnabled() bool {
	return s.Settings.IsEnabled()
}
