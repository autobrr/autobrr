// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package gotify

import (
	"context"
	"encoding/json"
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

	data := url.Values{}
	data.Set("message", message.Message)
	data.Set("title", message.Title)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "autobrr")
	// the header keeps the app token out of proxy and access logs, unlike the
	// ?token= query param
	req.Header.Set("X-Gotify-Key", c.config.Token)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "client request error")
	}

	defer sharedhttp.DrainAndClose(res)

	c.log.Trace().Int("status_code", res.StatusCode).Msg("response status")

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
			return errors.Wrap(statusError(res), "check gotify application token")
		}

		return statusError(res)
	}

	c.log.Debug().Msg("notification successfully sent to gotify")

	return nil
}

// statusError surfaces the error and description from the Gotify error
// payload, falling back to the raw body, and always reports the status code.
func statusError(res *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))

	var response struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"errorDescription"`
	}

	if err := json.Unmarshal(body, &response); err == nil && response.Error != "" {
		if response.ErrorDescription != "" {
			return errors.New("unexpected status: %d error: %s: %s", res.StatusCode, response.Error, response.ErrorDescription)
		}

		return errors.New("unexpected status: %d error: %s", res.StatusCode, response.Error)
	}

	if len(body) > 0 {
		return errors.New("unexpected status: %d body: %s", res.StatusCode, string(body))
	}

	return errors.New("unexpected status: %d", res.StatusCode)
}
