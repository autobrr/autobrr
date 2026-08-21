// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package ntfy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/avast/retry-go/v4"
	"github.com/rs/zerolog"
)

// waits are capped per attempt so two capped waits plus the requests still
// fit the caller's 30s send budget; a server demanding more than the cap is
// unrecoverable immediately
const maxRetryAfter = time.Second * 10

// ntfy generally rate-limits without a Retry-After header, so retries fall
// back to a short fixed delay. Shortened in tests.
var (
	defaultRetryAfter = time.Second * 5
	retryDelay        = time.Second * 5
)

// rateLimitError carries the server-directed wait so the retry delay
// function can honor it instead of the default fixed delay.
type rateLimitError struct {
	delay time.Duration
	msg   string
}

func (e *rateLimitError) Error() string {
	return e.msg
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
	err := retry.Do(func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.Host, strings.NewReader(message.Message))
		if err != nil {
			return retry.Unrecoverable(errors.Wrap(err, "could not create request"))
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

		switch res.StatusCode {
		case http.StatusOK:
			return nil

		case http.StatusUnauthorized, http.StatusForbidden:
			return retry.Unrecoverable(errors.Wrap(statusError(res), "could not authenticate with ntfy: check access token or username/password"))

		case http.StatusRequestEntityTooLarge:
			return retry.Unrecoverable(errors.Wrap(statusError(res), "message too large for ntfy"))

		case http.StatusTooManyRequests:
			statusErr := errors.Wrap(statusError(res), "rate limited by ntfy")

			delay := retryAfterDelay(res)
			if delay > maxRetryAfter {
				return retry.Unrecoverable(errors.Wrap(statusErr, "Retry-After %s exceeds the retry budget", delay))
			}

			return &rateLimitError{delay: delay, msg: statusErr.Error()}

		default:
			if res.StatusCode >= http.StatusInternalServerError {
				return statusError(res)
			}

			return retry.Unrecoverable(statusError(res))
		}
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

			return retry.FixedDelay(n, err, config)
		}),
	)
	if err != nil {
		return err
	}

	c.log.Debug().Msg("notification successfully sent to ntfy")

	return nil
}

type errorResponse struct {
	Code  int    `json:"code"`
	HTTP  int    `json:"http"`
	Error string `json:"error"`
}

// statusError surfaces the ntfy JSON error payload when present, falling back
// to the raw body, and always reports the status code.
func statusError(res *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))

	var errRes errorResponse
	if err := json.Unmarshal(body, &errRes); err == nil && errRes.Error != "" {
		return errors.New("unexpected status: %d error: %s code: %d", res.StatusCode, errRes.Error, errRes.Code)
	}

	if body := strings.TrimSpace(string(body)); body != "" {
		return errors.New("unexpected status: %d body: %s", res.StatusCode, body)
	}

	return errors.New("unexpected status: %d", res.StatusCode)
}

func retryAfterDelay(res *http.Response) time.Duration {
	value := res.Header.Get("Retry-After")
	if value == "" {
		return defaultRetryAfter
	}

	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	if at, err := http.ParseTime(value); err == nil {
		if until := time.Until(at); until > 0 {
			return until
		}
	}

	return defaultRetryAfter
}
