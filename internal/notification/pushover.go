// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"encoding/json"
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

func NewPushoverSender(log zerolog.Logger, settings *domain.Notification) Sender {
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

	// Use event-specific sound if available, otherwise fall back to global sound
	sound := ""
	if s.Settings.EventSounds != nil {
		if eventSound, ok := s.Settings.EventSounds[string(event)]; ok && eventSound != "" {
			sound = eventSound
		}
	}
	// Fall back to global sound if no event-specific sound
	if sound == "" && s.Settings.Sound != "" {
		sound = s.Settings.Sound
	}
	// Only send sound parameter if a sound is specified (empty means use user's default)
	if sound != "" {
		data.Set("sound", sound)
	}

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

	s.log.Debug().Msg("notification successfully sent to pushover")

	return nil
}

func (s *pushoverSender) IsEnabled() bool {
	return s.Settings.IsEnabled()
}

// GetSounds fetches available sounds from Pushover API
func GetPushoverSounds(apiToken string) (map[string]string, error) {
	if apiToken == "" {
		return nil, errors.New("api token is required")
	}

	url := fmt.Sprintf("https://api.pushover.net/1/sounds.json?token=%s", apiToken)

	client := &http.Client{
		Timeout:   time.Second * 30,
		Transport: sharedhttp.Transport,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not create request")
	}

	req.Header.Set("User-Agent", "autobrr")

	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "client request error")
	}

	defer sharedhttp.DrainAndClose(res)

	if res.StatusCode != http.StatusOK {
		limitedReader := io.LimitReader(res.Body, 4096)
		body, err := io.ReadAll(limitedReader)
		if err != nil {
			return nil, errors.Wrap(err, "could not read body")
		}
		return nil, errors.New("unexpected status: %v body: %v", res.StatusCode, string(body))
	}

	var response struct {
		Status int               `json:"status"`
		Sounds map[string]string `json:"sounds"`
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, errors.Wrap(err, "could not decode response")
	}

	if response.Status != 1 {
		return nil, errors.New("pushover API returned error status: %d", response.Status)
	}

	return response.Sounds, nil
}
