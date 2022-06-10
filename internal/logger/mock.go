package logger

import (
	"github.com/rs/zerolog"
	"io"
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
