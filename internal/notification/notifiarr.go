// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"bufio"
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

type notifiarrMessage struct {
	Event string               `json:"event"`
	Data  notifiarrMessageData `json:"data"`
}

type notifiarrMessageData struct {
	Subject        string                        `json:"subject"`
	Message        string                        `json:"message"`
	Event          domain.NotificationEvent      `json:"event"`
	ReleaseName    *string                       `json:"release_name,omitempty"`
	Filter         *string                       `json:"filter,omitempty"`
	Indexer        *string                       `json:"indexer,omitempty"`
	InfoHash       *string                       `json:"info_hash,omitempty"`
	Size           *uint64                       `json:"size,omitempty"`
	Status         *domain.ReleasePushStatus     `json:"status,omitempty"`
	Action         *string                       `json:"action,omitempty"`
	ActionType     *domain.ActionType            `json:"action_type,omitempty"`
	ActionClient   *string                       `json:"action_client,omitempty"`
	Rejections     []string                      `json:"rejections,omitempty"`
	Protocol       *domain.ReleaseProtocol       `json:"protocol,omitempty"`       // torrent, usenet
	Implementation *domain.ReleaseImplementation `json:"implementation,omitempty"` // irc, rss, api
	Timestamp      time.Time                     `json:"timestamp"`
}

type notifiarrSender struct {
	log      zerolog.Logger
	Settings *domain.Notification
	baseUrl  string

	httpClient *http.Client
}

func (s *notifiarrSender) Name() string {
	return "notifiarr"
}

func NewNotifiarrSender(log zerolog.Logger, settings *domain.Notification) domain.NotificationSender {
	return &notifiarrSender{
		log:      log.With().Str("sender", "notifiarr").Logger(),
		Settings: settings,
		baseUrl:  "https://notifiarr.com/api/v1/notification/autobrr",
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (s *notifiarrSender) Send(event domain.NotificationEvent, payload domain.NotificationPayload) error {
	m := notifiarrMessage{
		Event: string(event),
		Data:  s.buildMessage(payload),
	}

	jsonData, err := json.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "could not marshal json request for event: %v payload: %v", event, payload)
	}

	req, err := http.NewRequest(http.MethodPost, s.baseUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.Wrap(err, "could not create request for event: %v payload: %v", event, payload)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")
	req.Header.Set("X-API-Key", s.Settings.APIKey)

	res, err := s.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "client request error for event: %v payload: %v", event, payload)
	}

	defer res.Body.Close()

	s.log.Trace().Msgf("response status: %d", res.StatusCode)

	if res.StatusCode != http.StatusOK {
		body, err := io.ReadAll(bufio.NewReader(res.Body))
		if err != nil {
			return errors.Wrap(err, "could not read body for event: %v payload: %v", event, payload)
		}

		return errors.New("unexpected status: %v body: %v", res.StatusCode, string(body))
	}

	s.log.Debug().Msg("notification successfully sent to notifiarr")

	return nil
}

func (s *notifiarrSender) CanSend(event domain.NotificationEvent) bool {
	if s.isEnabled() && s.isEnabledEvent(event) {
		return true
	}
	return false
}

func (s *notifiarrSender) isEnabled() bool {
	if s.Settings.Enabled {
		if s.Settings.APIKey == "" {
			s.log.Warn().Msg("notifiarr missing api key")
			return false
		}

		return true
	}
	return false
}

func (s *notifiarrSender) isEnabledEvent(event domain.NotificationEvent) bool {
	for _, e := range s.Settings.Events {
		if e == string(event) {
			return true
		}
	}

	return false
}

func (s *notifiarrSender) buildMessage(payload domain.NotificationPayload) notifiarrMessageData {
	m := notifiarrMessageData{
		Event:     payload.Event,
		Timestamp: payload.Timestamp,
	}

	if payload.Subject != "" && payload.Message != "" {
		m.Subject = payload.Subject
		m.Message = payload.Message
	}
	if payload.ReleaseName != "" {
		m.ReleaseName = &payload.ReleaseName
	}
	if payload.Status != "" {
		m.Status = &payload.Status
	}
	if payload.Indexer != "" {
		m.Indexer = &payload.Indexer
	}
	if payload.Filter != "" {
		m.Filter = &payload.Filter
	}
	if payload.Action != "" || payload.ActionClient != "" {
		m.Action = &payload.Action

		if payload.ActionClient != "" {
			m.ActionClient = &payload.ActionClient
		}
	}
	if len(payload.Rejections) > 0 {
		m.Rejections = payload.Rejections
	}

	return m
}
