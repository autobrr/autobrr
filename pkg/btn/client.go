// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package btn

import (
	"context"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/jsonrpc"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

const DefaultURL = "https://api.broadcasthe.net/"

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
		url:         DefaultURL,
		rateLimiter: rate.NewLimiter(rate.Every(150*time.Hour), 1), // 150 rpcRequest every 1 hour
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
