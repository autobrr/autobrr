// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later
/*
Copyright (c) Microsoft Corporation.
Licensed under the MIT license.
*/

package sharedhttp

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var clients = map[string]*http.Client{}
var httpTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second, // default transport value
		KeepAlive: 30 * time.Second, // default transport value
	}).DialContext,
	ForceAttemptHTTP2:     true,             // default is true; since HTTP/2 multiplexes a single TCP connection. we'd want to use HTTP/1, which would use multiple TCP connections.
	MaxIdleConns:          100,              // default transport value
	MaxIdleConnsPerHost:   10,               // default is 2, so we want to increase the number to use establish more connections.
	IdleConnTimeout:       90 * time.Second, // default transport value
	TLSHandshakeTimeout:   10 * time.Second, // default transport value
	ExpectContinueTimeout: 1 * time.Second,  // default transport value
	ReadBufferSize:        65536,
	WriteBufferSize:       65536,
	TLSClientConfig: &tls.Config{
		MinVersion: tls.VersionTLS12,
	},
}

var insecureHTTPTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second, // default transport value
		KeepAlive: 30 * time.Second, // default transport value
	}).DialContext,
	ForceAttemptHTTP2:     true,             // default is true; since HTTP/2 multiplexes a single TCP connection. we'd want to use HTTP/1, which would use multiple TCP connections.
	MaxIdleConns:          100,              // default transport value
	MaxIdleConnsPerHost:   10,               // default is 2, so we want to increase the number to use establish more connections.
	IdleConnTimeout:       90 * time.Second, // default transport value
	TLSHandshakeTimeout:   10 * time.Second, // default transport value
	ExpectContinueTimeout: 1 * time.Second,  // default transport value
	ReadBufferSize:        65536,
	WriteBufferSize:       65536,
	TLSClientConfig: &tls.Config{
		InsecureSkipVerify: true,
	},
}

type HTTPOptions struct {
	Name     string
	Insecure bool
}

var lock sync.RWMutex

func GetClient(opts HTTPOptions) *http.Client {
	name := opts.Name
	if u, err := url.ParseRequestURI(name); err == nil && len(u.Host) != 0 {
		name = u.Host
	}

	lock.RLock()
	if c, ok := clients[name]; ok {
		lock.RUnlock()
		return c
	}
	lock.RUnlock()

	var c *http.Client
	if opts.Insecure {
		c = &http.Client{
			Transport: insecureHTTPTransport,
			Timeout:   time.Second * 120,
		}
	} else {
		c = &http.Client{
			Transport: httpTransport,
			Timeout:   time.Second * 120,
		}
	}

	lock.Lock()
	clients[name] = c
	lock.Unlock()
	return c
}
