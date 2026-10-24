// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"net"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStopAbortsProxyConnect(t *testing.T) {
	t.Parallel()

	// an HTTP proxy that accepts the connection and never answers CONNECT
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })

	accepted := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		accepted <- c
	}()

	network := domain.IrcNetwork{
		ID:       1,
		Name:     "proxied",
		Server:   "irc.example.test",
		Port:     6667,
		Nick:     "bot",
		Enabled:  true,
		UseProxy: true,
		ProxyId:  1,
		Proxy:    &domain.Proxy{ID: 1, Name: "http", Enabled: true, Type: domain.ProxyTypeHTTP, Addr: "http://" + ln.Addr().String()},
	}

	h := NewHandler(zerolog.Nop(), noopEventBus{}, &mockSSEServer{}, network, nil, nil)

	runErr := make(chan error, 1)
	go func() { runErr <- h.Run() }()

	select {
	case c := <-accepted:
		t.Cleanup(func() { c.Close() })
	case <-time.After(5 * time.Second):
		t.Fatal("proxy never saw the dial")
	}

	h.Stop()

	select {
	case err := <-runErr:
		assert.ErrorIs(t, err, clientManuallyDisconnected)
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not abort the proxy CONNECT in flight")
	}
}
