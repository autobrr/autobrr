// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package scheduler

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAnchoredSchedule_Next(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, time.November, 28, 14, 42, 13, 500_000_000, time.UTC)

	t.Run("first run is lastRun plus interval", func(t *testing.T) {
		t.Parallel()

		lastRun := now.Add(-5 * time.Minute)

		for i := range 100 {
			schedule := newAnchoredSchedule(15*time.Minute, lastRun, now, fmt.Sprintf("feed-%d", i))

			next := schedule.Next(now)
			assert.True(t, next.After(now), "next run must be in the future")
			// the pin may shift the anchor by up to half a wheel
			assert.WithinDuration(t, lastRun.Add(15*time.Minute), next, pinWheel(15*time.Minute)/2)
		}
	})

	t.Run("keeps the interval between runs", func(t *testing.T) {
		t.Parallel()

		schedule := newAnchoredSchedule(15*time.Minute, now.Add(-5*time.Minute), now, "feed-1")

		next := schedule.Next(now)
		for range 10 {
			following := schedule.Next(next)
			assert.Equal(t, 15*time.Minute, following.Sub(next))
			next = following
		}
	})

	t.Run("recovers cadence after a delayed wake", func(t *testing.T) {
		t.Parallel()

		schedule := newAnchoredSchedule(15*time.Minute, now.Add(-5*time.Minute), now, "feed-1")

		next := schedule.Next(now)
		following := schedule.Next(next.Add(90 * time.Second))
		assert.Equal(t, 15*time.Minute, following.Sub(next), "late wake must stay on the grid, not shift the phase")
	})

	t.Run("job that never ran catches up within a wheel", func(t *testing.T) {
		t.Parallel()

		for i := range 100 {
			schedule := newAnchoredSchedule(720*time.Minute, time.Time{}, now, fmt.Sprintf("feed-%d", i))

			next := schedule.Next(now)
			assert.True(t, next.After(now.Add(catchUpGrace)))
			assert.LessOrEqual(t, next.Sub(now), catchUpGrace+maxPinWheel, "catch-up must not wait a full interval")
		}
	})

	t.Run("overdue job catches up within a wheel", func(t *testing.T) {
		t.Parallel()

		lastRun := now.Add(-13 * time.Hour)
		schedule := newAnchoredSchedule(720*time.Minute, lastRun, now, "feed-1")

		next := schedule.Next(now)
		assert.True(t, next.After(now))
		assert.LessOrEqual(t, next.Sub(now), catchUpGrace+maxPinWheel)

		assert.Equal(t, 720*time.Minute, schedule.Next(next).Sub(next), "interval resumes from the catch-up run")
	})

	t.Run("lastRun in the future is clamped to now", func(t *testing.T) {
		t.Parallel()

		schedule := newAnchoredSchedule(15*time.Minute, now.Add(24*time.Hour), now, "feed-1")

		next := schedule.Next(now)
		assert.True(t, next.After(now))
		assert.WithinDuration(t, now.Add(15*time.Minute), next, pinWheel(15*time.Minute)/2)
	})

	t.Run("is stable for the same identifier", func(t *testing.T) {
		t.Parallel()

		lastRun := now.Add(-5 * time.Minute)

		a := newAnchoredSchedule(15*time.Minute, lastRun, now, "feed-7")
		b := newAnchoredSchedule(15*time.Minute, lastRun, now, "feed-7")

		assert.Equal(t, a.first, b.first)
		assert.Equal(t, a.Next(now), b.Next(now))
	})

	t.Run("restart re-anchors to the same phase", func(t *testing.T) {
		t.Parallel()

		schedule := newAnchoredSchedule(720*time.Minute, now.Add(-5*time.Hour), now, "feed-7")
		fireAt := schedule.Next(now)

		// process restarts a while after the fire; last_run was written seconds into the run
		rescheduled := newAnchoredSchedule(720*time.Minute, fireAt.Add(4*time.Second), fireAt.Add(2*time.Hour), "feed-7")

		assert.Equal(t, fireAt.Add(720*time.Minute), rescheduled.Next(fireAt.Add(2*time.Hour)))
	})

	t.Run("clamps intervals below a second", func(t *testing.T) {
		t.Parallel()

		for _, interval := range []time.Duration{0, -time.Minute, 500 * time.Millisecond} {
			schedule := newAnchoredSchedule(interval, time.Time{}, now, "feed-1")

			assert.Equal(t, time.Second, schedule.interval)
			assert.True(t, schedule.Next(now).After(now))
		}
	})

	t.Run("truncates sub-second intervals to whole seconds", func(t *testing.T) {
		t.Parallel()

		schedule := newAnchoredSchedule(90*time.Second+400*time.Millisecond, time.Time{}, now, "feed-1")

		assert.Equal(t, 90*time.Second, schedule.interval)
		assert.Zero(t, schedule.Next(now).Nanosecond())
	})
}

func TestPinWheel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		interval time.Duration
		wheel    time.Duration
	}{
		{time.Minute, time.Minute},
		{5 * time.Minute, time.Minute},
		{15 * time.Minute, time.Minute},
		{30 * time.Minute, 2 * time.Minute},
		{60 * time.Minute, 2 * time.Minute},
		{720 * time.Minute, 2 * time.Minute},
		{7 * time.Minute, time.Minute},
		{13 * time.Minute, time.Minute},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.wheel, pinWheel(tt.interval), "interval %s", tt.interval)
		assert.Zero(t, tt.interval%pinWheel(tt.interval), "wheel must divide the interval so the pin survives every run")
	}
}

// Feeds sharing an interval must not fire on the same second: cron reschedules every entry due in
// a tick from the same timestamp, so entries that collide once would stay locked together for the
// life of the process. The identifier-derived pin keeps them apart, even when their last runs
// coincide and even after a delayed wake that makes several feeds due in the same tick.
func TestAnchoredSchedule_SpreadsFeedsSharingAnInterval(t *testing.T) {
	t.Parallel()

	const (
		interval = 15 * time.Minute
		feeds    = 20
	)

	now := time.Date(2025, time.November, 28, 14, 42, 13, 0, time.UTC)
	lastRun := now.Add(-3 * time.Minute)

	fires := make(map[time.Time]int)
	afterMergedWake := make(map[time.Time]int)

	for i := range feeds {
		schedule := newAnchoredSchedule(interval, lastRun, now, fmt.Sprintf("feed-%d", i))

		fires[schedule.Next(now)]++

		// a delayed wake shared by every feed must not merge the fire times afterwards
		afterMergedWake[schedule.Next(now.Add(interval+time.Minute))]++
	}

	for _, m := range []map[time.Time]int{fires, afterMergedWake} {
		assert.Greater(t, len(m), feeds/2, "fires should be spread across distinct seconds")

		for fireTime, count := range m {
			assert.LessOrEqual(t, count, 3, "too many feeds landed on %s", fireTime)
		}
	}
}

// Overdue feeds at boot all catch up from the same moment; their pins must still spread them.
func TestAnchoredSchedule_SpreadsOverdueFeedsAtBoot(t *testing.T) {
	t.Parallel()

	const feeds = 30

	now := time.Date(2025, time.November, 28, 14, 42, 13, 0, time.UTC)
	lastRun := now.Add(-13 * time.Hour)

	fires := make(map[time.Time]int)

	for i := range feeds {
		schedule := newAnchoredSchedule(720*time.Minute, lastRun, now, fmt.Sprintf("feed-%d", i))
		fires[schedule.Next(now)]++
	}

	assert.Greater(t, len(fires), feeds*3/4, "catch-up fires should be spread across the wheel")

	for fireTime, count := range fires {
		assert.LessOrEqual(t, count, 2, "too many feeds landed on %s", fireTime)
	}
}
