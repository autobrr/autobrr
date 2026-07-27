// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package btn

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/jsonrpc"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

const DefaultURL = "https://api.broadcasthe.net/"

// btn answers 503 both for an invalid api key and for hitting the api too
// much, so we cannot tell the two apart. Either way the only useful response
// is to stop calling for a while instead of burning the remaining budget.
const throttleBackoff = 5 * time.Minute

type ApiClient interface {
	GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error)
	TestAPI(ctx context.Context) (bool, error)
}

type OptFunc func(*Client)

func WithUrl(url string) OptFunc {
	return func(c *Client) {
		c.url = url
	}
}

func WithHTTPClient(httpClient *http.Client) OptFunc {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

func WithLog(log zerolog.Logger) OptFunc {
	return func(c *Client) {
		c.log = log
	}
}

type Client struct {
	httpClient  *http.Client
	rpcClient   jsonrpc.Client
	rateLimiter *rate.Limiter
	APIKey      string
	url         string

	log zerolog.Logger

	m         sync.RWMutex
	throttled time.Time
}

// logger prefers the request-scoped logger from ctx, which carries trace_id
// and other correlation fields, over the static client logger.
func (c *Client) logger(ctx context.Context) *zerolog.Logger {
	if l := zerolog.Ctx(ctx); l.GetLevel() != zerolog.Disabled {
		return l
	}
	return &c.log
}

func NewClient(apiKey string, opts ...OptFunc) *Client {
	c := &Client{
		url: DefaultURL,
		// the api allows 150 calls per hour and answers 503 past that. the
		// burst rides on top of the refill, so the rate is set below the
		// ceiling to keep any one hour window at or under 150 calls.
		rateLimiter: rate.NewLimiter(rate.Every(time.Hour/145), 5),
		APIKey:      apiKey,
		httpClient: &http.Client{
			Timeout:   time.Second * 60,
			Transport: sharedhttp.Transport,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	c.rpcClient = jsonrpc.NewClientWithOpts(c.url, &jsonrpc.ClientOpts{
		Headers: map[string]string{
			"User-Agent": "autobrr",
		},
		HTTPClient: c.httpClient,
	})

	return c
}

func (c *Client) call(ctx context.Context, method string, params ...any) (*jsonrpc.RPCResponse, error) {
	if until := c.throttledUntil(); time.Now().Before(until) {
		return nil, errors.New("btn: api returned 503, not calling again until %s", until.Format(time.RFC3339))
	}

	start := time.Now()
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, errors.Wrap(err, "error waiting for ratelimiter")
	}

	if waited := time.Since(start); waited > time.Second {
		c.logger(ctx).Debug().Dur("waited", waited).Str("rpc_method", method).Msg("rate limiter delayed request")
	}

	res, err := c.rpcClient.CallCtx(ctx, method, params...)

	var httpErr *jsonrpc.HTTPError
	if errors.As(err, &httpErr) && httpErr.Code == http.StatusServiceUnavailable {
		until := c.throttle()
		c.logger(ctx).Warn().Str("rpc_method", method).Time("until", until).Msg("btn api returned 503, backing off. verify the api key if this persists")
	}

	return res, err
}

func (c *Client) throttledUntil() time.Time {
	c.m.RLock()
	defer c.m.RUnlock()

	return c.throttled
}

func (c *Client) throttle() time.Time {
	c.m.Lock()
	defer c.m.Unlock()

	c.throttled = time.Now().Add(throttleBackoff)

	return c.throttled
}
