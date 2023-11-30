// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/dustin/go-humanize"
	"github.com/rs/zerolog"
)

type gotifyMessage struct {
	Message string `json:"message"`
	Title   string `json:"title"`
}

type gotifySender struct {
	log      zerolog.Logger
	Settings domain.Notification
}

func NewGotifySender(log zerolog.Logger, settings domain.Notification) domain.NotificationSender {
	return &gotifySender{
		log:      log.With().Str("sender", "gotify").Logger(),
		Settings: settings,
	}
}

func (s *gotifySender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	m := gotifyMessage{
		Message: s.buildMessage(payload),
		Title:   s.buildTitle(event),
	}

	data := url.Values{}
	data.Set("message", m.Message)
	data.Set("title", m.Title)

	url := fmt.Sprintf("%v/message?token=%v", s.Settings.Host, s.Settings.Token)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(data.Encode()))
	if err != nil {
		s.log.Error().Err(err).Msgf("gotify client request error: %v", event)
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "autobrr")

	client := sharedhttp.GetClient(url, false)
	res, err := client.Do(req)
	if err != nil {
		s.log.Error().Err(err).Msgf("gotify client request error: %v", event)
		return errors.Wrap(err, "could not make request: %+v", req)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		s.log.Error().Err(err).Msgf("gotify client request error: %v", event)
		return errors.Wrap(err, "could not read data")
	}

	defer res.Body.Close()

	s.log.Trace().Msgf("gotify status: %v response: %v", res.StatusCode, string(body))

	if res.StatusCode != http.StatusOK {
		s.log.Error().Err(err).Msgf("gotify client request error: %v", string(body))
		return errors.New("bad status: %v body: %v", res.StatusCode, string(body))
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

func (s *gotifySender) buildMessage(payload domain.NotificationPayload) string {
	msg := ""

	if payload.Subject != "" && payload.Message != "" {
		msg += fmt.Sprintf("%v\n%v", payload.Subject, payload.Message)
	}
	if payload.ReleaseName != "" {
		msg += fmt.Sprintf("\nNew release: %v", payload.ReleaseName)
	}
	if payload.Size > 0 {
		msg += fmt.Sprintf("\nSize: %v", humanize.Bytes(payload.Size))
	}
	if payload.Status != "" {
		msg += fmt.Sprintf("\nStatus: %v", payload.Status.String())
	}
	if payload.Indexer != "" {
		msg += fmt.Sprintf("\nIndexer: %v", payload.Indexer)
	}
	if payload.Filter != "" {
		msg += fmt.Sprintf("\nFilter: %v", payload.Filter)
	}
	if payload.Action != "" {
		action := fmt.Sprintf("\nAction: %v Type: %v", payload.Action, payload.ActionType)
		if payload.ActionClient != "" {
			action += fmt.Sprintf(" Client: %v", payload.ActionClient)
		}
		msg += action
	}
	if len(payload.Rejections) > 0 {
		msg += fmt.Sprintf("\nRejections: %v", strings.Join(payload.Rejections, ", "))
	}

	return msg
}

func (s *gotifySender) buildTitle(event domain.NotificationEvent) string {
	title := ""

	switch event {
	case domain.NotificationEventAppUpdateAvailable:
		title = "Autobrr update available"
	case domain.NotificationEventPushApproved:
		title = "Push Approved"
	case domain.NotificationEventPushRejected:
		title = "Push Rejected"
	case domain.NotificationEventPushError:
		title = "Error"
	case domain.NotificationEventIRCDisconnected:
		title = "IRC Disconnected"
	case domain.NotificationEventIRCReconnected:
		title = "IRC Reconnected"
	case domain.NotificationEventTest:
		title = "Test"
	}

	return title
}
