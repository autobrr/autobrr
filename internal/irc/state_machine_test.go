// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

// TestOnStateEntryConnectingIsQuiet is a regression test for
// https://github.com/autobrr/autobrr/issues/2586: Connecting is the only state
// with no entry action and was missing from the onStateEntry switch, so every
// connect attempt fell through to the default branch and logged "invalid
// state" at error level.
func TestOnStateEntryConnectingIsQuiet(t *testing.T) {
	var buf bytes.Buffer
	h, _ := newTestHandler()
	h.log = zerolog.New(&buf)
	sm := NewConnectionStateMachine(h)

	sm.onStateEntry(StateConnecting)

	assert.Empty(t, buf.String())
}

// TestOnStateEntryUnknownStateLogs guards the default branch: it evaluates
// state.String() while building the log event, so an out-of-range value must
// stringify to a fallback instead of panicking on the name array index before
// the diagnostic is emitted.
func TestOnStateEntryUnknownStateLogs(t *testing.T) {
	var buf bytes.Buffer
	h, _ := newTestHandler()
	h.log = zerolog.New(&buf)
	sm := NewConnectionStateMachine(h)

	sm.onStateEntry(ConnectionState(99))

	assert.Contains(t, buf.String(), "invalid state")
	assert.Contains(t, buf.String(), "ConnectionState(99)")
}
