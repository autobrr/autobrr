// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package lunasea

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

type Config struct {
	WebhookURL string
	Name       string
}

type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Image string `json:"image,omitempty"`
}

type Client struct {
	log    zerolog.Logger
	config Config

	httpClient *http.Client
}

func NewSender(log zerolog.Logger, config Config) *Client {
	config.WebhookURL = rewriteWebhookURL(config.WebhookURL)

	return &Client{
		log:    log.With().Str("sender", "lunasea").Str("name", config.Name).Logger(),
		config: config,
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (c *Client) Name() string {
	return "lunasea"
}

var rxpWebhookPath = regexp.MustCompile(`/(radarr|sonarr|lidarr|tautulli|overseerr)/`)

// rewriteWebhookURL points the webhook at the custom module, which is what
// autobrr sends. It is not mentioned in the LunaSea docs, so users tend to
// paste one of the *arr URLs instead.
func rewriteWebhookURL(url string) string {
	return rxpWebhookPath.ReplaceAllString(url, "/custom/")
}

func (c *Client) SendMessage(ctx context.Context, message *Message) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "could not marshal message to json")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.WebhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", "application/json")
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

	c.log.Debug().Msg("notification successfully sent to lunasea")

	return nil
}
