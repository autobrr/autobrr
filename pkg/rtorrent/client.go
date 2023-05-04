// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package rtorrent

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/mrobinsn/go-rtorrent/rtorrent"
)

type Client struct {
	Name       string
	Hostname   string
	httpClient *http.Client

	// TLS skip cert validation
	TLSSkipVerify bool
	//// HTTP Basic auth username
	//BasicUser string
	//// HTTP Basic auth password
	//BasicPass string

	rt *rtorrent.RTorrent
}

type Config struct {
	Hostname string

	// TLS skip cert validation
	TLSSkipVerify bool

	// HTTP Basic auth username
	BasicUser string

	// HTTP Basic auth password
	BasicPass string
}

func NewClient(cfg Config) *Client {
	c := &Client{
		Hostname:      cfg.Hostname,
		TLSSkipVerify: cfg.TLSSkipVerify,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &authRoundTripper{
				BasicUser: cfg.BasicUser,
				BasicPass: cfg.BasicPass,
			},
		},
	}

	// create custom client
	c.rt = rtorrent.New(c.Hostname, true)
	c.rt.WithHTTPClient(c.httpClient)

	return c
}

func (c *Client) Add(url string, extraArgs ...*rtorrent.FieldValue) error {
	return c.rt.Add(url, extraArgs...)
}

func (c *Client) AddStopped(url string, extraArgs ...*rtorrent.FieldValue) error {
	return c.rt.AddStopped(url, extraArgs...)
}

type authRoundTripper struct {
	http.Transport
	BasicUser string
	BasicPass string
}

func (rt *authRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	rt.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	if rt.BasicUser != "" && rt.BasicPass != "" {
		r.SetBasicAuth(rt.BasicUser, rt.BasicPass)
	}

	return rt.Transport.RoundTrip(r)
}
