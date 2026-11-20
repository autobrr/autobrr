// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"sync"
	"time"
)

type timer interface {
	Stop() bool
}

type clock interface {
	Now() time.Time
	AfterFunc(d time.Duration, f func()) timer
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

type realTimer struct {
	t *time.Timer
}

func (r *realTimer) Stop() bool {
	if r == nil || r.t == nil {
		return false
	}
	return r.t.Stop()
}

func (realClock) AfterFunc(d time.Duration, f func()) timer {
	return &realTimer{t: time.AfterFunc(d, f)}
}

// mockClock is used for deterministic unit testing.
type mockClock struct {
	m      sync.Mutex
	now    time.Time
	timers []*mockTimer
}

type mockTimer struct {
	c        *mockClock
	fireAt   time.Time
	callback func()
	stopped  bool
}

func (mt *mockTimer) Stop() bool {
	mt.c.m.Lock()
	defer mt.c.m.Unlock()
	if mt.stopped {
		return false
	}
	mt.stopped = true
	return true
}

func newMockClock(initial time.Time) *mockClock {
	return &mockClock{now: initial}
}

func (mc *mockClock) Now() time.Time {
	mc.m.Lock()
	defer mc.m.Unlock()
	return mc.now
}

func (mc *mockClock) AfterFunc(d time.Duration, f func()) timer {
	mc.m.Lock()
	defer mc.m.Unlock()
	t := &mockTimer{
		c:        mc,
		fireAt:   mc.now.Add(d),
		callback: f,
	}
	mc.timers = append(mc.timers, t)
	return t
}

func (mc *mockClock) Advance(d time.Duration) {
	mc.m.Lock()
	mc.now = mc.now.Add(d)
	now := mc.now

	var toFire []func()
	var remaining []*mockTimer

	for _, t := range mc.timers {
		if !t.stopped && !now.Before(t.fireAt) {
			toFire = append(toFire, t.callback)
			t.stopped = true
		} else if !t.stopped {
			remaining = append(remaining, t)
		}
	}
	mc.timers = remaining
	mc.m.Unlock()

	for _, f := range toFire {
		f()
	}
}
