// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package scheduler

import (
	"hash/fnv"
	"time"
)

// catchUpGrace delays the catch-up run of a job that never ran or became overdue while the
// process was down, so a burst of overdue jobs at startup does not fire before boot settles.
const catchUpGrace = 10 * time.Second

// maxPinWheel caps the pin wheel: 2 minutes keeps the anchor within ±1 minute of
// lastRun+interval for every interval while leaving 120 slots, enough to make same-second
// collisions rare - and a colliding pair only costs two concurrent fetches.
const maxPinWheel = 2 * time.Minute

// anchoredSchedule runs a job every interval, anchored to the job's last run: the next fire is
// lastRun+interval instead of an arbitrary phase within the interval, so what the UI reports
// matches what a user expects from a refresh interval.
//
// Fires are pinned to a stable identifier-derived slot within a wheel that divides the interval,
// so every fire lands on the same slot. cron reschedules every entry due in a tick from the same
// timestamp, so jobs sharing an interval that ever collide on a fire second would otherwise stay
// locked together for the life of the process; distinct pins keep them permanently apart at the
// cost of shifting the anchor by at most half a wheel.
type anchoredSchedule struct {
	interval time.Duration
	first    time.Time
}

// newAnchoredSchedule computes the first fire from lastRun; a zero lastRun means the job never
// ran. Jobs that never ran or are already overdue catch up within a wheel of now instead of
// waiting up to a full interval.
func newAnchoredSchedule(interval time.Duration, lastRun time.Time, now time.Time, identifier string) anchoredSchedule {
	if interval < time.Second {
		interval = time.Second
	}

	// whole seconds only, cron fires are second-resolution
	interval -= interval % time.Second

	now = now.Truncate(time.Second)

	var wheel, pin time.Duration
	if interval%time.Minute == 0 {
		wheel = pinWheel(interval)

		h := fnv.New32a()
		h.Write([]byte(identifier))
		pin = time.Duration(uint64(h.Sum32())%uint64(wheel/time.Second)) * time.Second
	}

	if lastRun.After(now) {
		lastRun = now
	}

	if !lastRun.IsZero() {
		first := pinNearest(lastRun.Truncate(time.Second), wheel, pin).Add(interval)

		// the margin keeps a fire due at registration from being computed as already
		// past once cron picks the entry up, which would skip a whole interval
		if first.After(now.Add(time.Second)) {
			return anchoredSchedule{interval: interval, first: first}
		}
	}

	return anchoredSchedule{interval: interval, first: pinAfter(now.Add(catchUpGrace), wheel, pin)}
}

// pinWheel returns the largest whole-minute divisor of interval capped at maxPinWheel, the
// largest pin space in which every fire of the interval still lands on the same pin.
func pinWheel(interval time.Duration) time.Duration {
	minutes := interval / time.Minute

	for d := maxPinWheel / time.Minute; d > 1; d-- {
		if minutes%d == 0 {
			return d * time.Minute
		}
	}

	return time.Minute
}

// pinNearest returns the time closest to t that sits on pin within the wheel, at most half a
// wheel away.
func pinNearest(t time.Time, wheel, pin time.Duration) time.Time {
	if wheel == 0 {
		return t
	}

	pinned := t.Truncate(wheel).Add(pin)

	if d := pinned.Sub(t); d > wheel/2 {
		return pinned.Add(-wheel)
	} else if d <= -wheel/2 {
		return pinned.Add(wheel)
	}

	return pinned
}

// pinAfter returns the first time strictly after t that sits on pin within the wheel.
func pinAfter(t time.Time, wheel, pin time.Duration) time.Time {
	if wheel == 0 {
		return t
	}

	pinned := t.Truncate(wheel).Add(pin)

	for !pinned.After(t) {
		pinned = pinned.Add(wheel)
	}

	return pinned
}

// Next returns the first grid point first+k*interval strictly after t, so a delayed wake skips
// forward on the fixed grid instead of drifting the phase.
func (s anchoredSchedule) Next(t time.Time) time.Time {
	if s.first.After(t) {
		return s.first
	}

	elapsed := t.Sub(s.first)
	next := s.first.Add(elapsed - elapsed%s.interval)

	for !next.After(t) {
		next = next.Add(s.interval)
	}

	return next
}
