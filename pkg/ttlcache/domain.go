package ttlcache

import (
	"sync"
	"time"

	"github.com/titlerr/upgraderr/pkg/timecache"
)

const NoTTL time.Duration = 0
const DefaultTTL time.Duration = time.Nanosecond * 1

type Cache[K comparable, V any] struct {
	tc timecache.Cache
	l  sync.RWMutex
	o  Options[K, V]
	ch chan time.Time
	m  map[K]Item[V]
}

type Item[V any] struct {
	t time.Time
	d time.Duration
	v V
}

type Options[K comparable, V any] struct {
	defaultTTL        time.Duration
	defaultResolution time.Duration
	deallocationFunc  DeallocationFunc[K, V]
	noUpdateTime      bool
}

type DeallocationReason int

const (
	ReasonTimedOut = DeallocationReason(iota)
	ReasonDeleted  = DeallocationReason(iota)
)

type DeallocationFunc[K comparable, V any] func(key K, value V, reason DeallocationReason)
