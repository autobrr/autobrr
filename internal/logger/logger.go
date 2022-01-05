package logger

import (
	"io"
	"os"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
)

func Setup(cfg domain.Config, sse *sse.Server) {
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	switch cfg.LogLevel {
	case "INFO":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "DEBUG":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "ERROR":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "WARN":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "TRACE":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	}

	// setup console writer
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}

	writers := io.MultiWriter(consoleWriter)

	// if logPath set create file writer
	if cfg.LogPath != "" {
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.LogPath,
			MaxSize:    100, // megabytes
			MaxBackups: 3,
		}

		// overwrite writers
		writers = io.MultiWriter(consoleWriter, fileWriter)
	}

	log.Logger = log.Hook(&ServerSentEventHook{sse: sse})
	log.Logger = log.Output(writers)
}
