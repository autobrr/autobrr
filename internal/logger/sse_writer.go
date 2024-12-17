// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
)

type SSEWriter struct {
	// SSE
	SSE *sse.Server

	// TimeFormat specifies the format for timestamp in output.
	TimeFormat string

	// PartsOrder defines the order of parts in output.
	PartsOrder []string
}

func NewSSEWriter(sse *sse.Server, options ...func(w *SSEWriter)) SSEWriter {
	w := SSEWriter{
		SSE:        sse,
		TimeFormat: defaultTimeFormat,
		PartsOrder: defaultPartsOrder(),
	}

	for _, opt := range options {
		opt(&w)
	}

	return w
}

type LogMessage struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

func (m LogMessage) Bytes() ([]byte, error) {
	j, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (w SSEWriter) Write(p []byte) (n int, err error) {
	if w.SSE == nil {
		return 0, nil
	}

	var evt map[string]interface{}
	p = decodeIfBinaryToBytes(p)
	d := json.NewDecoder(bytes.NewReader(p))
	d.UseNumber()
	err = d.Decode(&evt)
	if err != nil {
		return n, fmt.Errorf("cannot decode event: %s", err)
	}

	buf := bytes.NewBuffer(make([]byte, 0, 100))
	defer func() {
		buf.Reset()
	}()

	for _, p := range w.PartsOrder {
		w.writePart(buf, evt, p)
	}

	w.writeFields(buf, evt)

	err = buf.WriteByte('\n')
	if err != nil {
		return n, err
	}

	m := LogMessage{
		//Time:    w.formatTime(evt),
		Time:    evt["time"].(string),
		Level:   w.formatLevel(evt),
		Message: buf.String(),
	}

	data, err := m.Bytes()
	if err != nil {
		return n, err
	}

	// publish too logs topic
	w.SSE.Publish("logs", &sse.Event{
		Data: data,
	})

	return len(p), err
}

// writeFields appends formatted key-value pairs to buf.
func (w SSEWriter) writeFields(buf *bytes.Buffer, evt map[string]interface{}) {
	var fields = make([]string, 0, len(evt))
	for field := range evt {

		switch field {
		case zerolog.LevelFieldName, zerolog.TimestampFieldName, zerolog.MessageFieldName, zerolog.CallerFieldName:
			continue
		}
		fields = append(fields, field)
	}
	sort.Strings(fields)

	// Write space only if something has already been written to the buffer, and if there are fields.
	if buf.Len() > 0 && len(fields) > 0 {
		buf.WriteByte(' ')
	}

	// Move the "error" field to the front
	ei := sort.Search(len(fields), func(i int) bool { return fields[i] >= zerolog.ErrorFieldName })
	if ei < len(fields) && fields[ei] == zerolog.ErrorFieldName {
		fields[ei] = ""
		fields = append([]string{zerolog.ErrorFieldName}, fields...)
		var xfields = make([]string, 0, len(fields))
		for _, field := range fields {
			if field == "" { // Skip empty fields
				continue
			}
			xfields = append(xfields, field)
		}
		fields = xfields
	}

	for i, field := range fields {
		var fn Formatter
		var fv Formatter

		if field == zerolog.ErrorFieldName {
			fn = defaultFormatErrFieldName()

			fv = defaultFormatErrFieldValue()
		} else {
			fn = defaultFormatFieldName()

			fv = defaultFormatFieldValue
		}

		buf.WriteString(fn(field))

		switch fValue := evt[field].(type) {
		case string:
			if needsQuote(fValue) {
				buf.WriteString(fv(strconv.Quote(fValue)))
			} else {
				buf.WriteString(fv(fValue))
			}
		case json.Number:
			buf.WriteString(fv(fValue))
		default:
			b, err := zerolog.InterfaceMarshalFunc(fValue)
			if err != nil {
				fmt.Fprintf(buf, "[error: %v]", err)
			} else {
				fmt.Fprint(buf, fv(b))
			}
		}

		if i < len(fields)-1 { // Skip space for last field
			buf.WriteByte(' ')
		}
	}
}

// writePart appends a formatted part to buf.
func (w SSEWriter) writePart(buf *bytes.Buffer, evt map[string]interface{}, p string) {
	var f Formatter

	switch p {
	case zerolog.LevelFieldName:
		f = defaultFormatLevel()

	case zerolog.TimestampFieldName:
		f = defaultFormatTimestamp(w.TimeFormat)

	case zerolog.MessageFieldName:
		f = defaultFormatMessage

	case zerolog.CallerFieldName:
		f = defaultFormatCaller()

	default:
		f = defaultFormatFieldValue
	}

	var s = f(evt[p])

	if len(s) > 0 {
		if buf.Len() > 0 {
			buf.WriteByte(' ') // Write space only if not the first part
		}
		buf.WriteString(s)
	}
}

// formatLevel format level to string
func (w SSEWriter) formatLevel(evt map[string]interface{}) string {
	var f Formatter

	f = defaultFormatLevel()

	var s = f(evt["level"])

	if len(s) > 0 {
		return s
	}

	return ""
}

// formatTime format time to string
func (w SSEWriter) formatTime(evt map[string]interface{}) string {
	var f Formatter

	f = defaultFormatTimestamp(w.TimeFormat)

	var s = f(evt["time"])

	if len(s) > 0 {
		return s
	}

	return ""
}

const (
	defaultTimeFormat = time.Kitchen
)

// Formatter transforms the input into a formatted string.
type Formatter func(interface{}) string

func decodeIfBinaryToBytes(in []byte) []byte {
	return in
}

// needsQuote returns true when the string s should be quoted in output.
func needsQuote(s string) bool {
	for i := range s {
		if s[i] < 0x20 || s[i] > 0x7e || s[i] == ' ' || s[i] == '\\' || s[i] == '"' {
			return true
		}
	}
	return false
}

// ----- DEFAULT FORMATTERS ---------------------------------------------------

func defaultPartsOrder() []string {
	return []string{
		//zerolog.TimestampFieldName,
		//zerolog.LevelFieldName,
		zerolog.CallerFieldName,
		zerolog.MessageFieldName,
	}
}

func defaultFormatTimestamp(timeFormat string) Formatter {
	if timeFormat == "" {
		timeFormat = defaultTimeFormat
	}
	return func(i interface{}) string {
		t := "<nil>"
		switch tt := i.(type) {
		case string:
			ts, err := time.ParseInLocation(zerolog.TimeFieldFormat, tt, time.Local)
			if err != nil {
				t = tt
			} else {
				t = ts.Local().Format(timeFormat)
			}
		case json.Number:
			i, err := tt.Int64()
			if err != nil {
				t = tt.String()
			} else {
				var sec, nsec int64

				switch zerolog.TimeFieldFormat {
				case zerolog.TimeFormatUnixNano:
					sec, nsec = 0, i
				case zerolog.TimeFormatUnixMicro:
					sec, nsec = 0, int64(time.Duration(i)*time.Microsecond)
				case zerolog.TimeFormatUnixMs:
					sec, nsec = 0, int64(time.Duration(i)*time.Millisecond)
				default:
					sec, nsec = i, 0
				}

				ts := time.Unix(sec, nsec)
				t = ts.Format(timeFormat)
			}
		}
		return t
	}
}

func defaultFormatLevel() Formatter {
	return func(i interface{}) string {
		var l string
		if ll, ok := i.(string); ok {
			switch ll {
			case zerolog.LevelTraceValue:
				l = "TRC"
			case zerolog.LevelDebugValue:
				l = "DBG"
			case zerolog.LevelInfoValue:
				l = "INF"
			case zerolog.LevelWarnValue:
				l = "WRN"
			case zerolog.LevelErrorValue:
				l = "ERR"
			case zerolog.LevelFatalValue:
				l = "FTL"
			case zerolog.LevelPanicValue:
				l = "PNC"
			default:
				l = ll
			}
		} else {
			if i == nil {
				l = "???"
			} else {
				l = strings.ToUpper(fmt.Sprintf("%s", i))[0:3]
			}
		}
		return l
	}
}

func defaultFormatCaller() Formatter {
	return func(i interface{}) string {
		var c string
		if cc, ok := i.(string); ok {
			c = cc
		}
		if len(c) > 0 {
			if cwd, err := os.Getwd(); err == nil {
				if rel, err := filepath.Rel(cwd, c); err == nil {
					c = rel
				}
			}
			c = c + " >"
		}
		return c
	}
}

func defaultFormatMessage(i interface{}) string {
	if i == nil {
		return ""
	}
	return fmt.Sprintf("%s", i)
}

func defaultFormatFieldName() Formatter {
	return func(i interface{}) string {
		return fmt.Sprintf("%s=", i)
	}
}

func defaultFormatFieldValue(i interface{}) string {
	return fmt.Sprintf("%s", i)
}

func defaultFormatErrFieldName() Formatter {
	return func(i interface{}) string {
		return fmt.Sprintf("%s=", i)
	}
}

func defaultFormatErrFieldValue() Formatter {
	return func(i interface{}) string {
		return fmt.Sprintf("%s=", i)
	}
}
