// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package logger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The sse writer reads the time field off every event, so both the root logger
// and the derived loggers services actually use have to carry a timestamp.
func TestNew_TimestampOnRootAndDerivedLoggers(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "autobrr.log")

	log := New(&domain.Config{LogLevel: "TRACE", Version: "1.0.0", LogPath: logPath}, nil)

	log.Info().Msg("root line")

	sub := log.With().Str("module", "irc").Logger()
	sub.Debug().Str("network", "example").Msg("derived line")

	b, err := os.ReadFile(logPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	require.Len(t, lines, 2)

	for _, line := range lines {
		var evt map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &evt), line)

		ts, ok := evt["time"].(string)
		assert.True(t, ok, "event must carry a string time field: %s", line)
		assert.NotEmpty(t, ts)
	}

	var derived map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &derived))
	assert.Equal(t, "irc", derived["module"])
	assert.Equal(t, "example", derived["network"])
}

func TestSetLevel(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "autobrr.log")

	log := New(&domain.Config{LogLevel: "INFO", Version: "1.0.0", LogPath: logPath}, nil)
	t.Cleanup(func() { SetLevel("TRACE") })

	log.Debug().Msg("filtered out")
	log.Info().Msg("kept")

	SetLevel("DEBUG")
	log.Debug().Msg("kept after level change")

	b, err := os.ReadFile(logPath)
	require.NoError(t, err)

	assert.NotContains(t, string(b), "filtered out")
	assert.Contains(t, string(b), "kept")
	assert.Contains(t, string(b), "kept after level change")
}
