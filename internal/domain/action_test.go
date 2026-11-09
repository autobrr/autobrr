// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAction_ReannounceSchedule(t *testing.T) {
	tests := []struct {
		name         string
		interval     int64
		maxAttempts  int64
		extraWaits   int
		wantInterval int
		wantAttempts int
	}{
		{name: "defaults", wantInterval: 7, wantAttempts: 50},
		{name: "within budget", interval: 10, maxAttempts: 30, wantInterval: 10, wantAttempts: 30},
		{name: "exactly the budget", interval: 30, maxAttempts: 30, wantInterval: 30, wantAttempts: 30},
		{name: "attempts capped", interval: 60, maxAttempts: 50, wantInterval: 60, wantAttempts: 15},
		{name: "default attempts capped", interval: 20, wantInterval: 20, wantAttempts: 45},
		{name: "interval longer than the budget", interval: 3600, maxAttempts: 3, wantInterval: 900, wantAttempts: 1},
		{name: "interval above 32-bit int", interval: 1 << 32, maxAttempts: 3, wantInterval: 900, wantAttempts: 1},
		{name: "attempts above 32-bit int", interval: 10, maxAttempts: 1<<31 + 1, wantInterval: 10, wantAttempts: 90},
		{name: "extra wait within budget", interval: 10, maxAttempts: 30, extraWaits: 1, wantInterval: 10, wantAttempts: 30},
		{name: "extra wait capped", interval: 60, maxAttempts: 50, extraWaits: 1, wantInterval: 60, wantAttempts: 14},
		{name: "extra wait with interval longer than the budget", interval: 3600, maxAttempts: 3, extraWaits: 1, wantInterval: 900, wantAttempts: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Action{ReAnnounceInterval: tt.interval, ReAnnounceMaxAttempts: tt.maxAttempts}

			interval, attempts := a.ReannounceSchedule(tt.extraWaits)
			assert.Equal(t, tt.wantInterval, interval)
			assert.Equal(t, tt.wantAttempts, attempts)
			assert.LessOrEqual(t, interval*(attempts+tt.extraWaits), int(ReannounceMaxDuration.Seconds()))
		})
	}
}
