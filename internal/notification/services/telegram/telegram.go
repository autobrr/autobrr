// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

const defaultHost = "https://api.telegram.org"

type Config struct {
	Host     string
	Token    string
	ChatID   string
	ThreadID int
	Name     string
}

// Message Reference: https://core.telegram.org/bots/api#sendmessage
type Message struct {
	ChatID          string `json:"chat_id"`
	Text            string `json:"text"`
	ParseMode       string `json:"parse_mode"`
	MessageThreadID int    `json:"message_thread_id,omitempty"`
}

type Client struct {
	log    zerolog.Logger
	config Config

	httpClient *http.Client
}

func NewSender(log zerolog.Logger, config Config) *Client {
	if config.Host == "" {
		config.Host = defaultHost
	}

	return &Client{
		log:    log.With().Str("sender", "telegram").Str("name", config.Name).Logger(),
		config: config,
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (c *Client) Name() string {
	return "telegram"
}

func (c *Client) SendMessage(ctx context.Context, message *Message) error {
	message.ChatID = c.config.ChatID
	message.MessageThreadID = c.config.ThreadID

	jsonData, err := json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "could not marshal message to json")
	}

	endpoint := c.config.Host + "/bot" + c.config.Token + "/sendMessage"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", "application/json")

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

	c.log.Debug().Msg("notification successfully sent to telegram")

	return nil
}
