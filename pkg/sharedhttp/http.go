// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
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

var Transport *http.Transport
var TransportTLSInsecure *http.Transport
var ProxyTransport *http.Transport
var Client *http.Client

func init() {
	// Initialize with default settings (no bind address)
	initTransports("")
}

// Init initializes the shared HTTP transports with an optional bind address.
// This should be called early in application startup after config is loaded.
// If bindAddress is empty, connections will use the default network interface.
func Init(bindAddress string) {
	initTransports(bindAddress)
}

// createDialer creates a net.Dialer with optional local address binding.
// If bindAddress is empty, the dialer will use the default behavior.
// If bindAddress is specified, outgoing connections will bind to that IP.
func createDialer(bindAddress string) *net.Dialer {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second, // default transport value
		KeepAlive: 30 * time.Second, // default transport value
	}

	if bindAddress != "" {
		dialer.LocalAddr = &net.TCPAddr{
			IP: net.ParseIP(bindAddress),
		}
	}

	return dialer
}

func initTransports(bindAddress string) {
	dialer := createDialer(bindAddress)

	Transport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
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

	TransportTLSInsecure = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
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

	ProxyTransport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
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

	Client = &http.Client{
		Timeout:   60 * time.Second,
		Transport: Transport,
	}
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

// DrainAndClose drains the response body and closes it to prevent connection leaks
func DrainAndClose(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
