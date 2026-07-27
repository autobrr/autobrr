// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package logger

import (
	"io"
	"os"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
)

// New builds the root logger. Pass the sse server to mirror log output to the
// web ui; every logger derived from the returned one inherits the writers, so
// there is no way to attach it afterwards.
func New(cfg *domain.Config, sseSrv sseServer) zerolog.Logger {
	SetLevel(cfg.LogLevel)

	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	writers := make([]io.Writer, 0, 3)

	// use pretty logging for dev only
	if cfg.Version == "dev" {
		writers = append(writers, zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	} else {
		writers = append(writers, os.Stderr)
	}

	if cfg.LogPath != "" {
		writers = append(writers, &lumberjack.Logger{
			Filename:   cfg.LogPath,
			MaxSize:    cfg.LogMaxSize, // megabytes
			MaxBackups: cfg.LogMaxBackups,
		})
	}

	if sseSrv != nil {
		writers = append(writers, NewSSEWriter(sseSrv))
	}

	return zerolog.New(io.MultiWriter(writers...)).With().Timestamp().Stack().Logger()
}

// SetLevel sets the level for every logger in the process. It is safe to call
// at runtime, which is what makes config reload work.
func SetLevel(level string) {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.DebugLevel
	}

	zerolog.SetGlobalLevel(lvl)
}

// Mock returns a logger that discards everything, for use in tests.
func Mock() zerolog.Logger {
	return zerolog.Nop()
}
