// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"testing"
	"time"
)

// TestModeAdds verifies the mode-string parser used by handleMode, including the
// cases a naive strings.Contains(modes, "+r") check gets wrong.
func TestModeAdds(t *testing.T) {
	cases := []struct {
		modes string
		flag  byte
		want  bool
	}{
		{"+r", 'r', true},
		{"-r", 'r', false},
		{"+ir", 'r', true},
		{"+nrt", 'r', true},  // combined add: substring "+r" is absent but r IS added
		{"+r-x", 'r', true},  // added, then a different mode removed
		{"+x-r", 'r', false}, // r removed
		{"+r-r", 'r', false}, // added then removed -> net removed
		{"-i+r", 'r', true},
		{"+i", 'r', false}, // no r at all
		{"", 'r', false},
		{"+B", 'B', true}, // bot-mode char
		{"-B", 'B', false},
	}
	for _, c := range cases {
		if got := modeAdds(c.modes, c.flag); got != c.want {
			t.Errorf("modeAdds(%q, %q) = %v, want %v", c.modes, string(c.flag), got, c.want)
		}
	}
}

// TestOnPartedSetsParted verifies parting a monitored channel moves it to the
// Parted state (previously it went to Idle, leaving Parted a dead state) and
// broadcasts it.
func TestOnPartedSetsParted(t *testing.T) {
	h, sse := newTestHandler()
	sm := addMonitoredChannel(h, "#chan", "")

	sm.OnParted()

	if got := sm.CurrentState(); got != ChannelStateParted {
		t.Fatalf("OnParted from Monitoring: state = %s, want Parted", got)
	}
	if !waitFor(func() bool { return sse.hasStateEvent("#chan", "Parted") }, time.Second) {
		t.Fatal("OnParted should broadcast the Parted state")
	}
}

// TestOnPartedIgnoredWhenNotMonitoring verifies OnParted only acts from Monitoring.
func TestOnPartedIgnoredWhenNotMonitoring(t *testing.T) {
	h, _ := newTestHandler()
	sm := newIdleSM(h, "#chan", "")

	sm.OnParted()

	if got := sm.CurrentState(); got != ChannelStateIdle {
		t.Fatalf("OnParted from Idle changed state to %s", got)
	}
}
