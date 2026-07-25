// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package scheduler

import (
	"fmt"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/assert"
)

func TestJitteredSchedule_Next(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, time.November, 28, 14, 42, 13, 500_000_000, time.UTC)

	t.Run("returns a time after now", func(t *testing.T) {
		t.Parallel()

		for i := 0; i < 100; i++ {
			schedule := newJitteredSchedule(15*time.Minute, fmt.Sprintf("feed-%d", i))
			assert.True(t, schedule.Next(now).After(now), "next run must be in the future")
		}
	})

	t.Run("keeps the interval between runs", func(t *testing.T) {
		t.Parallel()

		schedule := newJitteredSchedule(15*time.Minute, "feed-1")

		next := schedule.Next(now)
		for i := 0; i < 10; i++ {
			following := schedule.Next(next)
			assert.Equal(t, 15*time.Minute, following.Sub(next))
			next = following
		}
	})

	t.Run("offset stays within the interval", func(t *testing.T) {
		t.Parallel()

		for i := 0; i < 100; i++ {
			schedule := newJitteredSchedule(15*time.Minute, fmt.Sprintf("feed-%d", i))
			assert.GreaterOrEqual(t, schedule.offset, time.Duration(0))
			assert.Less(t, schedule.offset, 15*time.Minute)
			assert.LessOrEqual(t, schedule.Next(now).Sub(now), 15*time.Minute)
		}
	})

	t.Run("is stable for the same identifier", func(t *testing.T) {
		t.Parallel()

		a := newJitteredSchedule(15*time.Minute, "feed-7")
		b := newJitteredSchedule(15*time.Minute, "feed-7")

		assert.Equal(t, a.offset, b.offset)
		assert.Equal(t, a.Next(now), b.Next(now))
	})

	t.Run("clamps intervals below a second", func(t *testing.T) {
		t.Parallel()

		for _, interval := range []time.Duration{0, -time.Minute, 500 * time.Millisecond} {
			schedule := newJitteredSchedule(interval, "feed-1")

			assert.Equal(t, time.Second, schedule.interval)
			assert.True(t, schedule.Next(now).After(now))
		}
	})

	t.Run("truncates sub-second intervals to whole seconds", func(t *testing.T) {
		t.Parallel()

		schedule := newJitteredSchedule(90*time.Second+400*time.Millisecond, "feed-1")

		assert.Equal(t, 90*time.Second, schedule.interval)
		assert.Zero(t, schedule.Next(now).Nanosecond())
	})
}

// Feeds sharing an interval must not fire on the same second. cron.Every reschedules every entry
// due in a tick from the same timestamp, so without a per-feed offset they stay locked together
// for the life of the process.
func TestJitteredSchedule_SpreadsFeedsSharingAnInterval(t *testing.T) {
	t.Parallel()

	const (
		interval = 15 * time.Minute
		feeds    = 20
	)

	now := time.Date(2025, time.November, 28, 14, 42, 13, 0, time.UTC)

	constant := make(map[time.Time]int)
	jittered := make(map[time.Time]int)

	for i := 0; i < feeds; i++ {
		identifier := fmt.Sprintf("feed-%d", i)

		constant[cron.Every(interval).Next(now)]++
		jittered[newJitteredSchedule(interval, identifier).Next(now)]++
	}

	assert.Len(t, constant, 1, "cron.Every fires every feed on the same second")
	assert.Greater(t, len(jittered), feeds/2, "jittered feeds should be spread across the interval")

	for fireTime, count := range jittered {
		assert.LessOrEqual(t, count, 3, "too many feeds landed on %s", fireTime)
	}
}
