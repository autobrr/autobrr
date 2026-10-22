// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"net"

	"github.com/rs/zerolog"
)

func listen(log zerolog.Logger, addr string) (net.Listener, error) {
	var firstErr error
	for _, proto := range []string{"tcp", "tcp4", "tcp6"} {
		listener, err := net.Listen(proto, addr)
		if err == nil {
			return listener, nil
		}

		log.Error().Err(err).Str("protocol", proto).Str("addr", addr).Msg("could not listen")

		// the dual-stack attempt carries the real cause, later fallbacks
		// tend to fail with less useful errors like "no suitable address"
		if firstErr == nil {
			firstErr = err
		}
	}

	return nil, firstErr
}
