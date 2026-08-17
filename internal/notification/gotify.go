// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
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
	baseSender
	builder MessageBuilderPlainText

	httpClient *http.Client
}

func (s *gotifySender) Name() string {
	return "gotify"
}

func NewGotifySender(log zerolog.Logger, settings *domain.Notification) Sender {
	return &gotifySender{
		baseSender: newBaseSender("gotify", log, settings),
		builder:    MessageBuilderPlainText{},
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

	urlPath, err := url.JoinPath(s.Settings.Host, "message")
	if err != nil {
		return err
	}
	urlPath += "?token=" + url.QueryEscape(s.Settings.Token)

	req, err := http.NewRequest(http.MethodPost, urlPath, strings.NewReader(data.Encode()))
	if err != nil {
		return errors.Wrap(err, "could not create request for event: %v payload: %v", event, payload)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "autobrr")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "client request error for event: %v payload: %v", event, payload)
	}

	defer sharedhttp.DrainAndClose(res)

	s.log.Trace().Int("status_code", res.StatusCode).Msg("response status")

	if res.StatusCode != http.StatusOK {
		limitedReader := io.LimitReader(res.Body, 4096)
		body, err := io.ReadAll(limitedReader)
		if err != nil {
			return errors.Wrap(err, "could not read body for event: %v payload: %v", event, payload)
		}

		return errors.New("unexpected status: %v body: %v", res.StatusCode, string(body))
	}

	s.log.Debug().Msg("notification successfully sent to gotify")

	return nil
}
