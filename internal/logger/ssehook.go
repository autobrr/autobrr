package logger

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
)

type LogMessage struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

func (m LogMessage) ToJsonString() string {
	j, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(j)
}

type ServerSentEventHook struct {
	sse *sse.Server
}

func (h *ServerSentEventHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if h.sse != nil {
		// publish too logs topic
		logMsg := LogMessage{
			Time:    time.Now().Format(time.RFC3339),
			Level:   strings.ToUpper(level.String()),
			Message: msg,
		}

		h.sse.Publish("logs", &sse.Event{
			Data: []byte(logMsg.ToJsonString()),
		})
	}
}
