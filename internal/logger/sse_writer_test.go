// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package logger

import (
	"testing"

	"github.com/r3labs/sse/v2"
	"github.com/stretchr/testify/assert"
)

func TestSSEWriter_Write(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
	}{
		{
			name: "with timestamp",
			line: `{"level":"info","time":"2026-07-26T12:00:00Z","message":"hello"}`,
		},
		{
			// a missing timestamp used to panic on an unchecked type assertion
			name: "without timestamp",
			line: `{"level":"info","message":"hello"}`,
		},
		{
			name: "timestamp of unexpected type",
			line: `{"level":"info","time":12345,"message":"hello"}`,
		},
		{
			name: "with fields",
			line: `{"level":"error","time":"2026-07-26T12:00:00Z","module":"irc","error":"boom","message":"hello"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := sse.New()
			srv.CreateStreamWithOpts(StreamLogs, sse.StreamOpts{MaxEntries: 10, AutoReplay: true})
			defer srv.Close()

			w := NewSSEWriter(srv)

			n, err := w.Write([]byte(tt.line))

			assert.NoError(t, err)
			assert.Equal(t, len(tt.line), n, "writer must report the whole line consumed")
		})
	}
}

func TestSSEWriter_WriteWithoutServer(t *testing.T) {
	t.Parallel()

	w := NewSSEWriter(nil)

	line := []byte(`{"level":"info","time":"2026-07-26T12:00:00Z","message":"hello"}`)

	n, err := w.Write(line)

	assert.NoError(t, err)
	assert.Equal(t, len(line), n, "a nil server must not look like a short write")
}
