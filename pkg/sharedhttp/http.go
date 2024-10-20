// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package sharedhttp

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

var Transport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second, // default transport value
		KeepAlive: 30 * time.Second, // default transport value
	}).DialContext,
	ForceAttemptHTTP2:     true,              // default is true; since HTTP/2 multiplexes a single TCP connection.
	MaxIdleConns:          100,               // default transport value
	MaxIdleConnsPerHost:   10,                // default is 2, so we want to increase the number to use establish more connections.
	IdleConnTimeout:       90 * time.Second,  // default transport value
	ResponseHeaderTimeout: 120 * time.Second, // servers can respond slowly - this should fix some portion of releases getting stuck as pending.
	TLSHandshakeTimeout:   10 * time.Second,  // default transport value
	ExpectContinueTimeout: 1 * time.Second,   // default transport value
	ReadBufferSize:        65536,
	WriteBufferSize:       65536,
	TLSClientConfig: &tls.Config{
		MinVersion: tls.VersionTLS12,
	},
}

var TransportTLSInsecure = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second, // default transport value
		KeepAlive: 30 * time.Second, // default transport value
	}).DialContext,
	ForceAttemptHTTP2:     true,              // default is true; since HTTP/2 multiplexes a single TCP connection.
	MaxIdleConns:          100,               // default transport value
	MaxIdleConnsPerHost:   10,                // default is 2, so we want to increase the number to use establish more connections.
	IdleConnTimeout:       90 * time.Second,  // default transport value
	ResponseHeaderTimeout: 120 * time.Second, // servers can respond slowly - this should fix some portion of releases getting stuck as pending.
	TLSHandshakeTimeout:   10 * time.Second,  // default transport value
	ExpectContinueTimeout: 1 * time.Second,   // default transport value
	ReadBufferSize:        65536,
	WriteBufferSize:       65536,
	TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true,
	},
}

var Client = &http.Client{
	Timeout:   60 * time.Second,
	Transport: Transport,
}

type MagnetRoundTripper struct{}

func (rt *MagnetRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme == "magnet" {
		responseBody := r.URL.String()
		respReader := io.NopCloser(strings.NewReader(responseBody))

		resp := &http.Response{
			Status:        http.StatusText(http.StatusOK),
			StatusCode:    http.StatusOK,
			Body:          respReader,
			ContentLength: int64(len(responseBody)),
			Header: map[string][]string{
				"Content-Type": {"text/plain"},
				"Location":     {responseBody},
			},
			Proto:      "HTTP/2.0",
			ProtoMajor: 2,
		}

		return resp, nil
	}

	return Transport.RoundTrip(r)
}

var MagnetTransport = &MagnetRoundTripper{}
