// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package scheduler

import (
	"hash/fnv"
	"time"
)

// jitteredSchedule runs a job every interval, phase-shifted by a stable per-job offset.
//
// cron.Every rounds every fire time to the second and reschedules all entries due in a tick from
// the same timestamp, so jobs registered together with the same interval fire on the same second
// and stay locked together for the life of the process. The offset spreads them across the
// interval instead. Each job still runs once per interval, so this costs no freshness.
type jitteredSchedule struct {
	interval time.Duration
	offset   time.Duration
}

// newJitteredSchedule derives the offset from identifier so a job keeps its slot across restarts.
func newJitteredSchedule(interval time.Duration, identifier string) jitteredSchedule {
	if interval < time.Second {
		interval = time.Second
	}

	// whole seconds only, to match cron.Every
	interval -= interval % time.Second

	h := fnv.New32a()
	h.Write([]byte(identifier))

	seconds := uint64(interval / time.Second)

	return jitteredSchedule{
		interval: interval,
		offset:   time.Duration(uint64(h.Sum32())%seconds) * time.Second,
	}
}

func (s jitteredSchedule) Next(t time.Time) time.Time {
	next := t.Truncate(s.interval).Add(s.offset)

	for !next.After(t) {
		next = next.Add(s.interval)
	}

	return next
}
