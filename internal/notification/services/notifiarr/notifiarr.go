// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notifiarr

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

const apiEndpoint = "https://notifiarr.com/api/v1/notification/autobrr"

type MessageSender interface {
	SendMessage(ctx context.Context, message *Message) error
}

type Config struct {
	APIKey string
	Name   string
}

type Message struct {
	Event string      `json:"event"`
	Data  MessageData `json:"data"`
}

type MessageData struct {
	Subject        string    `json:"subject"`
	Message        string    `json:"message"`
	Event          string    `json:"event"`
	ReleaseName    *string   `json:"release_name,omitempty"`
	Filter         *string   `json:"filter,omitempty"`
	Indexer        *string   `json:"indexer,omitempty"`
	InfoHash       *string   `json:"info_hash,omitempty"`
	Size           *uint64   `json:"size,omitempty"`
	Status         *string   `json:"status,omitempty"`
	Action         *string   `json:"action,omitempty"`
	ActionType     *string   `json:"action_type,omitempty"`
	ActionClient   *string   `json:"action_client,omitempty"`
	Rejections     []string  `json:"rejections,omitempty"`
	Protocol       *string   `json:"protocol,omitempty"`       // torrent, usenet
	Implementation *string   `json:"implementation,omitempty"` // irc, rss, api
	Timestamp      time.Time `json:"timestamp"`
}

type Client struct {
	log    zerolog.Logger
	config Config

	httpClient *http.Client
}

func NewSender(log zerolog.Logger, config Config) *Client {
	return &Client{
		log:    log.With().Str("sender", "notifiarr").Str("name", config.Name).Logger(),
		config: config,
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: sharedhttp.Transport,
		},
	}
}

func (c *Client) Name() string {
	return "notifiarr"
}

func (c *Client) SendMessage(ctx context.Context, message *Message) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "could not marshal message to json")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.Wrap(err, "could not create request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")
	req.Header.Set("X-API-Key", c.config.APIKey)

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

	c.log.Debug().Msg("notification successfully sent to notifiarr")

	return nil
}
