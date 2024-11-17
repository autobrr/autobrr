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
	de time.Duration
	ch chan time.Duration
	m  map[K]item[V]
}

type item[V any] struct {
	t time.Time
	d time.Duration
	v V
}

type Options struct {
	DefaultTTL time.Duration
}

func New[K comparable, V any](options Options) *Cache[K, V] {
	c := Cache[K, V]{
		de: options.DefaultTTL,
		ch: make(chan time.Duration, 1000),
		m:  make(map[K]item[V]),
	}

	go c.startExpirations()
	return &c
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	it, ok := c.get(key)
	if !ok {
		return *new(V), ok
	}

	if !it.t.IsZero() && c.getDuration(it.d) != it.t {
		c.set(key, it)
	}

	return it.v, ok
}

func (c *Cache[K, V]) Set(key K, value V, duration time.Duration) bool {
	if c.de == NoTTL && duration == DefaultTTL {
		duration = NoTTL
	}

	c.set(key, item[V]{v: value, d: duration})
	return true
}

func (c *Cache[K, V]) Delete(key K) {
	c.delete(key)
}

func (c *Cache[K, V]) get(key K) (item[V], bool) {
	c.l.RLock()
	defer c.l.RUnlock()
	v, ok := c.m[key]

	if !ok {
		return item[V]{}, ok
	}

	return v, ok
}

func (c *Cache[K, V]) set(key K, it item[V]) {
	it.t = c.getDuration(it.d)

	c.l.Lock()
	defer c.l.Unlock()
	c.m[key] = it
	c.ch <- it.d
}

func (c *Cache[K, V]) delete(key K) {
	c.l.Lock()
	defer c.l.Unlock()
	delete(c.m, key)
}

func (c *Cache[K, V]) getDuration(d time.Duration) time.Time {
	switch d {
	case NoTTL:
	case DefaultTTL:
		return c.tc.Now().Add(c.de)
	default:
		return c.tc.Now().Add(d)
	}

	return time.Time{}
}

func (c *Cache[K, V]) startExpirations() {
	timer := time.NewTimer(1 * time.Second)
	timer.Stop() // wasteful, but makes the loop cleaner because this is initialized.

	var timeSleep time.Time
	for {
		select {
		case d, ok := <-c.ch:
			if !ok {
				timer.Stop()
				return
			} else if d == NoTTL {
				continue
			} else if d == DefaultTTL {
				d = c.de
			}

			t := c.tc.Now()
			if timeSleep.IsZero() || timeSleep.After(t.Add(d)) {
				timeSleep = t.Add(d)
				if !timer.Reset(d) {
					timer = time.NewTimer(d)
				}
			}

		case <-timer.C:
			timer.Stop()
			c.expire()
			timeSleep = time.Time{}
		}
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

		delete(c.m, k)
	}

	if !soon.IsZero() {
		c.ch <- soon.Sub(t)
	}
}

func (c *Cache[K, V]) Close() {
	c.l.Lock() // deadlock on reentry.
	close(c.ch)
	c.ch = nil
}
