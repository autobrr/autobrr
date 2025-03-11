// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package logger

import (
	"io"
	"os"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger interface
type Logger interface {
	Log() *zerolog.Event
	Fatal() *zerolog.Event
	Err(err error) *zerolog.Event
	Error() *zerolog.Event
	Warn() *zerolog.Event
	Info() *zerolog.Event
	Trace() *zerolog.Event
	Debug() *zerolog.Event
	With() zerolog.Context
	RegisterSSEWriter(sse *sse.Server)
	SetLogLevel(level string)
	Printf(format string, v ...interface{})
	Print(v ...interface{})
}

// DefaultLogger default logging controller
type DefaultLogger struct {
	log     zerolog.Logger
	writers []io.Writer
	level   zerolog.Level
}

func New(cfg *domain.Config) Logger {
	l := &DefaultLogger{
		writers: make([]io.Writer, 0),
		level:   zerolog.DebugLevel,
	}

	// set log level
	l.SetLogLevel(cfg.LogLevel)

	// use pretty logging for dev only
	if cfg.Version == "dev" {
		// setup console writer
		consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}

		l.writers = append(l.writers, consoleWriter)
	} else {
		// default to stderr
		l.writers = append(l.writers, os.Stderr)
	}

	if cfg.LogPath != "" {
		l.writers = append(l.writers,
			&lumberjack.Logger{
				Filename:   cfg.LogPath,
				MaxSize:    cfg.LogMaxSize, // megabytes
				MaxBackups: cfg.LogMaxBackups,
			},
		)
	}

	// set some defaults
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	// init new logger
	l.log = zerolog.New(io.MultiWriter(l.writers...)).With().Stack().Logger()

	return l
}

func (l *DefaultLogger) RegisterSSEWriter(sse *sse.Server) {
	w := NewSSEWriter(sse)
	l.writers = append(l.writers, w)
	l.log = zerolog.New(io.MultiWriter(l.writers...)).With().Stack().Logger()
}

func (l *DefaultLogger) SetLogLevel(level string) {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(lvl)
}

// Log log something at fatal level.
func (l *DefaultLogger) Log() *zerolog.Event {
	return l.log.Log().Timestamp()
}

// Fatal log something at fatal level. This will panic!
func (l *DefaultLogger) Fatal() *zerolog.Event {
	return l.log.Fatal().Timestamp()
}

// Error log something at Error level
func (l *DefaultLogger) Error() *zerolog.Event {
	return l.log.Error().Timestamp()
}

// Err log something at Err level
func (l *DefaultLogger) Err(err error) *zerolog.Event {
	return l.log.Err(err).Timestamp()
}

// Warn log something at warning level.
func (l *DefaultLogger) Warn() *zerolog.Event {
	return l.log.Warn().Timestamp()
}

// Info log something at fatal level.
func (l *DefaultLogger) Info() *zerolog.Event {
	return l.log.Info().Timestamp()
}

// Debug log something at debug level.
func (l *DefaultLogger) Debug() *zerolog.Event {
	return l.log.Debug().Timestamp()
}

// Trace log something at fatal level. This will panic!
func (l *DefaultLogger) Trace() *zerolog.Event {
	return l.log.Trace().Timestamp()
}

// Print sends a log event using debug level and no extra field.
// Arguments are handled in the manner of fmt.Print.
func (l *DefaultLogger) Print(v ...interface{}) {
	l.log.Print(v...)
}

// Printf sends a log event using debug level and no extra field.
// Arguments are handled in the manner of fmt.Printf.
func (l *DefaultLogger) Printf(format string, v ...interface{}) {
	l.log.Printf(format, v...)
}

// With log with context
func (l *DefaultLogger) With() zerolog.Context {
	return l.log.With().Timestamp()
}
