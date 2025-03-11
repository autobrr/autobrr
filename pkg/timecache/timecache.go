// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package timecache

import (
	"sync"
	"time"
)

type Cache struct {
	t time.Time
	o Options
	m sync.RWMutex
}
type Options struct {
	round time.Duration
}

func New(o Options) *Cache {
	c := Cache{
		o: o,
	}
	return &c
}
func (t *Cache) Now() time.Time {
	t.m.RLock()
	if !t.t.IsZero() {
		defer t.m.RUnlock()
		return t.t
	}
	t.m.RUnlock()
	return t.update()
}
func (t *Cache) update() time.Time {
	t.m.Lock()
	defer t.m.Unlock()
	if !t.t.IsZero() {
		return t.t
	}
	var d time.Duration
	if t.o.round > time.Nanosecond {
		d = t.o.round
	} else {
		d = time.Second * 1
	}
	t.t = time.Now().Round(d)
	go func(duration time.Duration) {
		if t.o.round > time.Nanosecond {
			duration = t.o.round / 2
		}
		time.Sleep(duration)
		t.reset()
	}(d)
	return t.t
}
func (t *Cache) reset() {
	t.m.Lock()
	defer t.m.Unlock()
	t.t = time.Time{}
}
func (o Options) Round(d time.Duration) Options {
	o.round = d
	return o
}
