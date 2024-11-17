package ttlcache

import (
	"sync"
	"time"

	"github.com/autobrr/autobrr/pkg/timecache"
)

const NoTTL time.Duration = 0
const DefaultTTL time.Duration = -1

type Cache[K comparable, V any] struct {
	tc timecache.Cache
	l  sync.RWMutex
	o  Options[K, V]
	ch chan time.Duration
	m  map[K]item[V]
}

type item[V any] struct {
	t time.Time
	d time.Duration
	v V
}

type Options[K comparable, V any] struct {
	defaultTTL       time.Duration
	deallocationFunc DeallocationFunc[K, V]
}

type DeallocationReason int

const (
	ReasonTimedOut = DeallocationReason(iota)
	ReasonDeleted  = DeallocationReason(iota)
)

type DeallocationFunc[K comparable, V any] func(key K, value V, reason DeallocationReason)
