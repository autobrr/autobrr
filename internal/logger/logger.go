package logger

import (
	"io"
	"os"
	"time"

	"github.com/autobrr/autobrr/internal/config"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

func Setup(cfg config.Cfg) {
	zerolog.TimeFieldFormat = time.RFC3339

	switch cfg.LogLevel {
	case "INFO":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "DEBUG":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "ERROR":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "WARN":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
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

	log.Logger = log.Output(writers)

	log.Print("Starting autobrr")
	log.Printf("Log-level: %v", cfg.LogLevel)
}
