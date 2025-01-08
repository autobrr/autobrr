// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package transmission

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/hekmon/transmissionrpc/v3"
)

type Config struct {
	UserAgent     string
	CustomClient  *http.Client
	Username      string
	Password      string
	TLSSkipVerify bool
	Timeout       time.Duration
}

func New(endpoint *url.URL, cfg *Config) (*transmissionrpc.Client, error) {
	ct := &customTransport{
		Username:      cfg.Username,
		Password:      cfg.Password,
		TLSSkipVerify: cfg.TLSSkipVerify,
	}

	extra := &transmissionrpc.Config{
		CustomClient: &http.Client{
			Transport: ct,
			Timeout:   time.Second * 60,
		},
		UserAgent: cfg.UserAgent,
	}

	return transmissionrpc.New(endpoint, extra)
}

type customTransport struct {
	Username      string
	Password      string
	TLSSkipVerify bool
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	dt := sharedhttp.Transport
	if t.TLSSkipVerify {
		dt.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	r := req.Clone(req.Context())

	if t.Username != "" && t.Password != "" {
		r.SetBasicAuth(t.Username, t.Password)
	}

	return dt.RoundTrip(r)
}
