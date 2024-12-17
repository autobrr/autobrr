// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package logger

import (
	"io"

	"github.com/rs/zerolog"
)

func Mock() Logger {
	l := &DefaultLogger{
		writers: make([]io.Writer, 0),
		level:   zerolog.Disabled,
	}

	// init new logger
	l.log = zerolog.New(io.MultiWriter(l.writers...)).With().Stack().Logger()

	return l
}
