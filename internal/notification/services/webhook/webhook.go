// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
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
	URL     string
	Method  string
	Headers string
	Name    string
}

// Message carries the event type for the X-Autobrr-Event header and the
// payload to marshal as the request body.
type Message struct {
	Event   string
	Payload any
}

type Client struct {
	log    zerolog.Logger
	config Config

	headers    map[string]string
	httpClient *http.Client
}

func NewSender(log zerolog.Logger, config Config) *Client {
	if config.Method == "" {
		config.Method = http.MethodPost
	}

	return &Client{
		log:     log.With().Str("sender", "webhook").Str("name", config.Name).Logger(),
		config:  config,
		headers: parseHeaders(config.Headers),
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (c *Client) Name() string {
	return "webhook"
}

// parseHeaders splits the user supplied "KEY=value,KEY2=value2" form into
// headers, skipping anything without a value.
func parseHeaders(headers string) map[string]string {
	if headers == "" {
		return nil
	}

	parsed := make(map[string]string)
	for header := range strings.SplitSeq(headers, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(header), "=")
		if !ok {
			continue
		}

		parsed[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	return parsed
}

func (c *Client) SendMessage(ctx context.Context, message *Message) error {
	jsonData, err := json.Marshal(message.Payload)
	if err != nil {
		return errors.Wrap(err, "could not marshal message to json")
	}

	req, err := http.NewRequestWithContext(ctx, c.config.Method, c.config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")
	req.Header.Set("X-Autobrr-Event", message.Event)

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "client request error")
	}

	defer sharedhttp.DrainAndClose(res)

	c.log.Trace().Int("status_code", res.StatusCode).Msg("response status")

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
		if err != nil {
			return errors.Wrap(err, "could not read response body")
		}

		return errors.New("unexpected status: %v body: %v", res.StatusCode, string(body))
	}

	c.log.Debug().Str("event", message.Event).Msg("notification successfully sent to webhook")

	return nil
}
