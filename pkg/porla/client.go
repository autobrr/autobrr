// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package porla

import (
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/jsonrpc"
	"github.com/autobrr/autobrr/pkg/sharedhttp"
)

var (
	DefaultTimeout = 60 * time.Second
)

type Client struct {
	cfg       Config
	rpcClient jsonrpc.Client
	http      *http.Client

	log      *log.Logger
	Name     string
	Hostname string
	timeout  time.Duration
}

type Config struct {
	Log       *log.Logger
	Hostname  string
	AuthToken string

	// HTTP Basic auth username
	BasicUser string

	// HTTP Basic auth password
	BasicPass string

	Timeout int

	// TLS skip cert validation
	TLSSkipVerify bool
}

func NewClient(cfg Config) *Client {
	c := &Client{
		cfg:     cfg,
		log:     log.New(io.Discard, "", log.LstdFlags),
		timeout: DefaultTimeout,
	}

	// override logger if we pass one
	if cfg.Log != nil {
		c.log = cfg.Log
	}

	if cfg.Timeout > 0 {
		c.timeout = time.Duration(cfg.Timeout) * time.Second
	}

	httpClient := &http.Client{
		Timeout:   c.timeout,
		Transport: sharedhttp.Transport,
	}

	if cfg.TLSSkipVerify {
		httpClient.Transport = sharedhttp.TransportTLSInsecure
	}

	c.http = httpClient

	token := cfg.AuthToken

	if !strings.HasPrefix(token, "Bearer ") {
		token = "Bearer " + token
	}

	c.rpcClient = jsonrpc.NewClientWithOpts(cfg.Hostname+"/api/v1/jsonrpc", &jsonrpc.ClientOpts{
		Headers: map[string]string{
			"X-Porla-Token": token,
		},
		HTTPClient: httpClient,
		BasicUser:  cfg.BasicUser,
		BasicPass:  cfg.BasicPass,
	})

	return c
}
