// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"net"
	"testing"

	"github.com/autobrr/autobrr/internal/auth"
	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/domain"

	"github.com/alexedwards/scs/v2"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func newLifecycleTestServer(port int) *Server {
	cfg := &domain.Config{Host: "127.0.0.1", Port: port, BaseURL: "/", BaseURLModeLegacy: true}

	return NewServer(Deps{
		Log:            zerolog.Nop(),
		SSE:            sse.New(),
		Config:         &config.AppConfig{Config: cfg},
		SessionManager: scs.New(),
		OIDCService:    auth.NewOIDCService(zerolog.Nop(), cfg),
	})
}

func TestServer_CloseListener(t *testing.T) {
	t.Run("releases the port when serve never ran", func(t *testing.T) {
		srv := newLifecycleTestServer(0)
		require.NoError(t, srv.Listen())

		port := srv.listener.Addr().(*net.TCPAddr).Port

		require.NoError(t, srv.CloseListener(context.Background()))

		again := newLifecycleTestServer(port)
		require.NoError(t, again.Listen())
		require.NoError(t, again.CloseListener(context.Background()))
	})

	t.Run("no-op after shutdown", func(t *testing.T) {
		srv := newLifecycleTestServer(0)
		require.NoError(t, srv.Listen())

		served := make(chan error, 1)
		go func() {
			served <- srv.Serve()
		}()

		require.NoError(t, srv.Shutdown(context.Background()))
		require.NoError(t, <-served)
		require.NoError(t, srv.CloseListener(context.Background()))
	})
}
