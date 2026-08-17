// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package pushover

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

const (
	messagesEndpoint = "https://api.pushover.net/1/messages.json"
	soundsEndpoint   = "https://api.pushover.net/1/sounds.json"
)

// PriorityEmergency messages are repeated until the user acknowledges them, so
// they must carry a retry interval and an expiry.
const PriorityEmergency = 2

type MessageSender interface {
	SendMessage(ctx context.Context, message *Message) error
}

type Config struct {
	Token string
	User  string
	Name  string
}

type Message struct {
	Title     string
	Message   string
	Priority  int32
	Sound     string
	Timestamp time.Time
	HTML      bool
}

type Client struct {
	log    zerolog.Logger
	config Config

	httpClient *http.Client
}

func NewSender(log zerolog.Logger, config Config) *Client {
	return &Client{
		log:    log.With().Str("sender", "pushover").Str("name", config.Name).Logger(),
		config: config,
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (c *Client) Name() string {
	return "pushover"
}

func (c *Client) SendMessage(ctx context.Context, message *Message) error {
	data := url.Values{}
	data.Set("token", c.config.Token)
	data.Set("user", c.config.User)
	data.Set("message", message.Message)
	data.Set("title", message.Title)
	data.Set("priority", strconv.Itoa(int(message.Priority)))
	data.Set("timestamp", strconv.FormatInt(message.Timestamp.Unix(), 10))

	if message.HTML {
		data.Set("html", "1")
	}

	// an empty sound falls back to the sound the user picked in the app
	if message.Sound != "" {
		data.Set("sound", message.Sound)
	}

	if message.Priority == PriorityEmergency {
		data.Set("expire", "3600")
		data.Set("retry", "60")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, messagesEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "autobrr")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "client request error")
	}

	defer sharedhttp.DrainAndClose(res)

	c.log.Trace().Int("status_code", res.StatusCode).Msg("response status")

	if res.StatusCode != http.StatusOK {
		body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
		if err != nil {
			return errors.Wrap(err, "could not read response body")
		}

		return errors.New("unexpected status: %v body: %v", res.StatusCode, string(body))
	}

	c.log.Debug().Msg("notification successfully sent to pushover")

	return nil
}

// GetSounds fetches the sounds available to the application token.
func GetSounds(ctx context.Context, apiToken string) (map[string]string, error) {
	if apiToken == "" {
		return nil, errors.New("api token is required")
	}

	endpoint := soundsEndpoint + "?token=" + url.QueryEscape(apiToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not create request")
	}

	req.Header.Set("User-Agent", "autobrr")

	client := &http.Client{
		Timeout:   time.Second * 30,
		Transport: sharedhttp.Transport,
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "client request error")
	}

	defer sharedhttp.DrainAndClose(res)

	if res.StatusCode != http.StatusOK {
		body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
		if err != nil {
			return nil, errors.Wrap(err, "could not read response body")
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
		return nil, errors.New("pushover api returned error status: %d", response.Status)
	}

	return response.Sounds, nil
}
