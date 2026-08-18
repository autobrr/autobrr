// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package events

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/tozd/go/errors"
)

// The caller field is stamped with a hardcoded frame skip, so it silently
// starts pointing at the bus itself if the Emit<Event>/On<Event> wrappers gain
// or lose a level of indirection.
func TestEventBus_CallerPointsAtCallSite(t *testing.T) {
	var buf bytes.Buffer

	bus := NewEventBus(zerolog.New(&buf).Level(zerolog.TraceLevel), t.Context())

	unregister := bus.OnAppUpdate(func(ctx context.Context, event AppUpdateEvent) errors.E {
		return nil
	})
	defer unregister()

	bus.EmitAppUpdate(AppUpdateEvent{Event: Event{Type: ApplicationUpdate}, NewVersion: "v1.7.0"})

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.NotEmpty(t, lines)

	for _, line := range lines {
		var entry struct {
			Caller  string `json:"caller"`
			Message string `json:"message"`
		}
		require.NoError(t, json.Unmarshal([]byte(line), &entry))

		assert.Containsf(t, entry.Caller, "eventbus_test.go", "%q logged caller %q", entry.Message, entry.Caller)
	}
}
