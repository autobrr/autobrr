// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package migrations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseSQLiteTimestamp(t *testing.T) {
	t.Parallel()

	want := time.Date(2026, 7, 26, 23, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		raw         string
		wantChanged bool
		wantOK      bool
	}{
		{name: "rfc3339_offset", raw: "2026-07-27T01:00:00+02:00", wantChanged: true, wantOK: true},
		{name: "rfc3339_utc", raw: "2026-07-26T23:00:00Z", wantChanged: true, wantOK: true},
		{name: "default_current_timestamp", raw: "2026-07-26 23:00:00", wantOK: true},
		{name: "time_string", raw: "2026-07-27 01:00:00 +0200 CEST", wantChanged: true, wantOK: true},
		{name: "time_string_monotonic", raw: "2026-07-27 01:00:00 +0200 CEST m=+3.5", wantChanged: true, wantOK: true},
		{name: "driver_layout_offset", raw: "2026-07-27 01:00:00+02:00", wantChanged: true, wantOK: true},
		{name: "driver_layout_utc", raw: "2026-07-26 23:00:00+00:00", wantOK: true},
		{name: "garbage", raw: "not a timestamp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, ok := parseSQLiteTimestamp(tt.raw)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantChanged, changed)

			if tt.wantOK {
				assert.True(t, want.Equal(got), "got %s", got)
			}
		})
	}
}
