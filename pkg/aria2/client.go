// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package aria2

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/jsonrpc"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

var DefaultTimeout = 60 * time.Second

// Client talks to an aria2 daemon over its JSON-RPC 2.0 interface.
type Client struct {
	cfg       Config
	rpcClient jsonrpc.Client

	log zerolog.Logger
}

type Config struct {
	// Host is the daemon address. The /jsonrpc path is appended when missing.
	Host string

	// Secret is the value of aria2s --rpc-secret. Empty when the daemon runs without one.
	Secret string

	// TLS skip cert validation
	TLSSkipVerify bool

	// HTTP Basic auth username
	BasicUser string

	// HTTP Basic auth password
	BasicPass string

	Timeout int
	Log     zerolog.Logger
}

func NewClient(cfg Config) (*Client, error) {
	endpoint, err := BuildEndpoint(cfg.Host)
	if err != nil {
		return nil, err
	}

	timeout := DefaultTimeout
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}

	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: sharedhttp.Transport,
	}

	if cfg.TLSSkipVerify {
		httpClient.Transport = sharedhttp.TransportTLSInsecure
	}

	c := &Client{
		cfg: cfg,
		log: cfg.Log,
		rpcClient: jsonrpc.NewClientWithOpts(endpoint, &jsonrpc.ClientOpts{
			HTTPClient: httpClient,
			BasicUser:  cfg.BasicUser,
			BasicPass:  cfg.BasicPass,
		}),
	}

	return c, nil
}

// BuildEndpoint turns a user supplied host into the aria2 JSON-RPC endpoint.
// aria2 documents the endpoint as ws://host:6800/jsonrpc, but the same handler
// answers over http, so the websocket schemes are mapped rather than rejected.
func BuildEndpoint(host string) (string, error) {
	if host == "" {
		return "", errors.New("aria2: missing host")
	}

	// without a scheme host:port parses as scheme:opaque
	if !strings.Contains(host, "://") {
		host = "http://" + host
	}

	u, err := url.Parse(host)
	if err != nil {
		return "", errors.Wrap(err, "aria2: could not parse host url: %s", host)
	}

	switch u.Scheme {
	case "ws":
		u.Scheme = "http"
	case "wss":
		u.Scheme = "https"
	case "http", "https":
	default:
		return "", errors.New("aria2: unsupported scheme %s in host url: %s", u.Scheme, host)
	}

	if u.Host == "" {
		return "", errors.New("aria2: missing host in url: %s", host)
	}

	if !strings.HasSuffix(strings.TrimSuffix(u.Path, "/"), "/jsonrpc") {
		path, err := url.JoinPath(u.Path, "/jsonrpc")
		if err != nil {
			return "", errors.Wrap(err, "aria2: could not build endpoint path: %s", host)
		}
		u.Path = path
	}

	return u.String(), nil
}

// params prefixes the call arguments with the secret token. aria2 takes it as
// the first positional parameter of every method instead of a header, and
// rejects the call when a token is sent to a daemon started without one.
func (c *Client) params(args ...any) []any {
	params := make([]any, 0, len(args)+1)

	if c.cfg.Secret != "" {
		params = append(params, "token:"+c.cfg.Secret)
	}

	return append(params, args...)
}

func (c *Client) call(ctx context.Context, method string, args ...any) (*jsonrpc.RPCResponse, error) {
	c.log.Trace().Str("method", method).Msg("aria2 rpc call")

	response, err := c.rpcClient.CallCtx(ctx, method, c.params(args...))
	if err != nil {
		return nil, errors.Wrap(err, "could not call %s", method)
	}

	if response.Error != nil {
		return nil, errors.Wrap(response.Error, "error calling %s", method)
	}

	return response, nil
}
