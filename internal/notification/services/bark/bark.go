// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package bark

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/meta"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

type Config struct {
	Host      string
	DeviceKey string
	Name      string
}

type Message struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body"`
	Sound string `json:"sound,omitempty"`
	Group string `json:"group,omitempty"`
	Level string `json:"level,omitempty"`
	Icon  string `json:"icon,omitempty"`
	URL   string `json:"url,omitempty"`
}

type pushRequest struct {
	DeviceKey string `json:"device_key"`
	*Message
}

type Client struct {
	log    zerolog.Logger
	config Config

	httpClient *http.Client
	userAgent  string
}

func NewSender(log zerolog.Logger, config Config) *Client {
	return &Client{
		log:    log.With().Str("sender", "bark").Str("name", config.Name).Logger(),
		config: config,
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
		userAgent: meta.GetUserAgent(),
	}
}

func (c *Client) Name() string {
	return "bark"
}

func (c *Client) SendMessage(ctx context.Context, message *Message) error {
	endpoint, err := url.JoinPath(c.config.Host, "push")
	if err != nil {
		return errors.Wrap(err, "could not build url from host: '%s'", c.config.Host)
	}

	jsonData, err := json.Marshal(pushRequest{DeviceKey: c.config.DeviceKey, Message: message})
	if err != nil {
		return errors.Wrap(err, "could not marshal message to json")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", c.userAgent)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "client request error")
	}

	defer sharedhttp.DrainAndClose(res)

	c.log.Trace().Int("status_code", res.StatusCode).Msg("response status")

	if res.StatusCode != http.StatusOK {
		switch res.StatusCode {
		case http.StatusBadRequest:
			// bark-server answers 400 when the key is not registered on that server
			return errors.Wrap(statusError(res), "check bark device key")
		case http.StatusTeapot:
			// bark-server answers failed basic auth with 418 instead of 401
			return errors.Wrap(statusError(res), "check bark server basic auth credentials")
		}

		return statusError(res)
	}

	c.log.Debug().Msg("notification successfully sent to bark")

	return nil
}

// statusError surfaces the message from the bark-server error payload,
// falling back to the raw body, and always reports the status code.
func statusError(res *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))

	var response struct {
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &response); err == nil && response.Message != "" {
		return errors.New("unexpected status: %d error: %s", res.StatusCode, response.Message)
	}

	if body := strings.TrimSpace(string(body)); body != "" {
		return errors.New("unexpected status: %d body: %s", res.StatusCode, body)
	}

	return errors.New("unexpected status: %d", res.StatusCode)
}
