// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package ntfy

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

type MessageSender interface {
	SendMessage(ctx context.Context, message *Message) error
}

type Config struct {
	Host     string
	Token    string
	Username string
	Password string
	Name     string
}

type Message struct {
	Title    string
	Message  string
	Priority int32
	Tags     string
}

type Client struct {
	log    zerolog.Logger
	config Config

	httpClient *http.Client
}

func NewSender(log zerolog.Logger, config Config) *Client {
	return &Client{
		log:    log.With().Str("sender", "ntfy").Str("name", config.Name).Logger(),
		config: config,
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (c *Client) Name() string {
	return "ntfy"
}

func (c *Client) SendMessage(ctx context.Context, message *Message) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.Host, strings.NewReader(message.Message))
	if err != nil {
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", "autobrr")
	req.Header.Set("Title", message.Title)

	if message.Priority > 0 {
		req.Header.Set("Priority", strconv.Itoa(int(message.Priority)))
	}
	if message.Tags != "" {
		req.Header.Set("Tags", message.Tags)
	}

	if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	} else if c.config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.Token)
	}

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

	c.log.Debug().Msg("notification successfully sent to ntfy")

	return nil
}
