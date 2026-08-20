// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package gotify

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

type Config struct {
	Host  string
	Token string
	Name  string
}

type Message struct {
	Title   string
	Message string
}

type Client struct {
	log    zerolog.Logger
	config Config

	httpClient *http.Client
}

func NewSender(log zerolog.Logger, config Config) *Client {
	return &Client{
		log:    log.With().Str("sender", "gotify").Str("name", config.Name).Logger(),
		config: config,
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (c *Client) Name() string {
	return "gotify"
}

func (c *Client) SendMessage(ctx context.Context, message *Message) error {
	endpoint, err := url.JoinPath(c.config.Host, "message")
	if err != nil {
		return errors.Wrap(err, "could not build url from host: '%s'", c.config.Host)
	}
	endpoint += "?token=" + url.QueryEscape(c.config.Token)

	data := url.Values{}
	data.Set("message", message.Message)
	data.Set("title", message.Title)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
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

	c.log.Debug().Msg("notification successfully sent to gotify")

	return nil
}
