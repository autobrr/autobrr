// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notifiarr

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/avast/retry-go/v4"
	"github.com/rs/zerolog"
)

const apiEndpoint = "https://notifiarr.com/api/v1/notification/autobrr"

// waits are capped per attempt so two capped waits plus the requests still
// fit the caller's 30s send budget; a server demanding more than the cap is
// unrecoverable immediately
const maxRetryAfter = time.Second * 10

// Shortened in tests.
var (
	defaultRetryAfter = time.Second * 5
	retryDelay        = time.Millisecond * 500
)

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

type apiResponse struct {
	Result  string `json:"result"`
	Details struct {
		Response string `json:"response"`
	} `json:"details"`
}

// rateLimitError carries the server-directed wait so the retry delay
// function can honor it instead of the default backoff.
type rateLimitError struct {
	delay time.Duration
	msg   string
}

func (e *rateLimitError) Error() string {
	return e.msg
}

type Client struct {
	log    zerolog.Logger
	config Config

	endpoint   string
	httpClient *http.Client
}

func NewSender(log zerolog.Logger, config Config) *Client {
	return &Client{
		log:      log.With().Str("sender", "notifiarr").Str("name", config.Name).Logger(),
		config:   config,
		endpoint: apiEndpoint,
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

	err = retry.Do(func() error {
		return c.send(ctx, jsonData)
	},
		retry.Context(ctx),
		retry.LastErrorOnly(true),
		retry.Attempts(3),
		retry.Delay(retryDelay),
		retry.DelayType(func(n uint, err error, config *retry.Config) time.Duration {
			var rle *rateLimitError
			if errors.As(err, &rle) && rle.delay > 0 {
				return rle.delay
			}

			return retry.BackOffDelay(n, err, config)
		}),
	)
	if err != nil {
		return err
	}

	c.log.Debug().Msg("notification successfully sent to notifiarr")

	return nil
}

func (c *Client) send(ctx context.Context, jsonData []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return retry.Unrecoverable(errors.Wrap(err, "could not create request"))
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

	switch {
	case res.StatusCode == http.StatusOK:
		return nil

	case res.StatusCode == http.StatusTooManyRequests:
		statusErr := statusError(res)

		delay := retryAfterDelay(res)
		if delay > maxRetryAfter {
			return retry.Unrecoverable(errors.Wrap(statusErr, "server demanded %s wait which exceeds the retry budget", delay))
		}

		return &rateLimitError{delay: delay, msg: statusErr.Error()}

	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		return retry.Unrecoverable(errors.Wrap(statusError(res), "check notifiarr api key"))

	// 408 is transient, let it fall through to the retryable default
	case res.StatusCode >= 400 && res.StatusCode < 500 && res.StatusCode != http.StatusRequestTimeout:
		return retry.Unrecoverable(statusError(res))

	default:
		return statusError(res)
	}
}

// statusError surfaces the Notifiarr JSON error response when present, falling
// back to the raw body, and always reports the status code.
func statusError(res *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))

	var apiErr apiResponse
	_ = json.Unmarshal(body, &apiErr)

	detail := apiErr.Details.Response
	if detail == "" {
		detail = string(bytes.TrimSpace(body))
	}

	if detail != "" {
		return errors.New("unexpected status: %d: %s", res.StatusCode, detail)
	}

	return errors.New("unexpected status: %d", res.StatusCode)
}

func retryAfterDelay(res *http.Response) time.Duration {
	if header := res.Header.Get("Retry-After"); header != "" {
		if seconds, err := strconv.Atoi(header); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}

	return defaultRetryAfter
}
