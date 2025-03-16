// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package ttlcache

import (
	"github.com/autobrr/autobrr/pkg/timecache"
	"sync"
	"time"
)

const NoTTL time.Duration = 0
const DefaultTTL time.Duration = time.Nanosecond * 1

type Cache[K comparable, V any] struct {
	ch chan time.Time
	m  map[K]Item[V]
	tc timecache.Cache
	o  Options[K, V]
	l  sync.RWMutex
}
type Item[V any] struct {
	t time.Time
	v V
	d time.Duration
}
type Options[K comparable, V any] struct {
	deallocationFunc  DeallocationFunc[K, V]
	defaultTTL        time.Duration
	defaultResolution time.Duration
	noUpdateTime      bool
}
type DeallocationReason int

const (
	ReasonTimedOut = DeallocationReason(iota)
	ReasonDeleted  = DeallocationReason(iota)
)

type DeallocationFunc[K comparable, V any] func(key K, value V, reason DeallocationReason)
