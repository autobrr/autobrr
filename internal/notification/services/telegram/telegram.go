// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/avast/retry-go/v4"
	"github.com/rs/zerolog"
)

const defaultHost = "https://api.telegram.org"

// waits are capped per attempt so two capped waits plus the requests still
// fit the caller's 30s send budget; a server demanding more than the cap is
// unrecoverable immediately
const maxRetryAfter = time.Second * 10

// Shortened in tests.
var (
	defaultRetryAfter = time.Second * 2
	retryDelay        = time.Millisecond * 500
)

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

type apiResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
	Parameters  struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters"`
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

	// join path segments so a malformed host fails here instead of inside
	// net/http, where the resulting url.Error would embed the bot token
	endpoint, err := url.JoinPath(c.config.Host, "bot"+c.config.Token, "sendMessage")
	if err != nil {
		return errors.New("could not build telegram api url: check custom host: %s", c.config.Host)
	}

	err = retry.Do(func() error {
		return c.send(ctx, endpoint, jsonData)
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

	c.log.Debug().Msg("notification successfully sent to telegram")

	return nil
}

func (c *Client) send(ctx context.Context, endpoint string, jsonData []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonData))
	if err != nil {
		// do not wrap: a url.Error here would embed the token-bearing endpoint
		return retry.Unrecoverable(errors.New("could not create request: invalid telegram api url"))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "autobrr")

	res, err := c.httpClient.Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return errors.Wrap(urlErr.Err, "client request error")
		}

		return errors.Wrap(err, "client request error")
	}

	defer sharedhttp.DrainAndClose(res)

	c.log.Trace().Int("status_code", res.StatusCode).Msg("response status")

	switch res.StatusCode {
	case http.StatusOK:
		return nil

	case http.StatusTooManyRequests:
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))

		var apiErr apiResponse
		_ = json.Unmarshal(body, &apiErr)

		statusErr := apiErr.statusError(res.StatusCode, body)

		delay := retryAfterDelay(res, &apiErr)
		if delay > maxRetryAfter {
			return retry.Unrecoverable(errors.Wrap(statusErr, "server demanded %s wait which exceeds the retry budget", delay))
		}

		return &rateLimitError{delay: delay, msg: statusErr.Error()}

	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return retry.Unrecoverable(errors.Wrap(statusError(res), "check bot token and chat id"))

	default:
		return statusError(res)
	}
}

func (r *apiResponse) statusError(statusCode int, body []byte) error {
	detail := r.Description
	if detail == "" {
		detail = string(bytes.TrimSpace(body))
	}

	if detail != "" {
		return errors.New("unexpected status: %d: %s", statusCode, detail)
	}

	return errors.New("unexpected status: %d", statusCode)
}

// statusError surfaces the Telegram JSON error description when present,
// falling back to the raw body, and always reports the status code.
func statusError(res *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))

	var apiErr apiResponse
	_ = json.Unmarshal(body, &apiErr)

	return apiErr.statusError(res.StatusCode, body)
}

func retryAfterDelay(res *http.Response, apiErr *apiResponse) time.Duration {
	if apiErr.Parameters.RetryAfter > 0 {
		return time.Duration(apiErr.Parameters.RetryAfter) * time.Second
	}

	if header := res.Header.Get("Retry-After"); header != "" {
		if seconds, err := strconv.Atoi(header); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}

	return defaultRetryAfter
}
