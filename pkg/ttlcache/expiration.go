// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package ttlcache

import (
	"time"
)

func (c *Cache[K, V]) startExpirations() {
	timer := time.NewTimer(1 * time.Second)
	stopTimer(timer) // wasteful, but makes the loop cleaner because this is initialized.
	defer stopTimer(timer)

	var timeSleep time.Time
	for {
		select {
		case t, ok := <-c.ch:
			if !ok {
				return
			} else if t.IsZero() {
				continue
			}

			if timeSleep.IsZero() || timeSleep.After(t) {
				timeSleep = t
				restartTimer(timer, timeSleep.Sub(c.tc.Now()))
			}

		case <-timer.C:
			stopTimer(timer)
			c.expire()
			timeSleep = time.Time{}
		}
	}
}

func restartTimer(t *time.Timer, d time.Duration) {
	stopTimer(t)
	t.Reset(d)
}

func stopTimer(t *time.Timer) {
	t.Stop()

	// go < 1.23 returns stale values on expired timers.
	if len(t.C) != 0 {
		<-t.C
	}
}

func (c *Cache[K, V]) expire() {
	t := c.tc.Now()
	var soon time.Time

	c.l.Lock()
	defer c.l.Unlock()
	for k, v := range c.m {
		if v.t.IsZero() {
			continue
		} else if v.t.After(t) {
			if soon.IsZero() || soon.After(v.t) {
				soon = v.t
			}
			continue
		}

		c.deleteUnsafe(k, v, ReasonTimedOut)
	}

	if !soon.IsZero() { // wake-up feedback loop
		go func(s time.Time) { // we need to release the lock, if the input pipeline has exceeded the wakeup budget.
			defer func() {
				_ = recover() // if the channel is closed, this doesn't matter on shutdown because this is expected.
			}()
			c.ch <- s
		}(soon)
	}
}
