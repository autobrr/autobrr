package ttlcache

import (
	"time"
)

func New[K comparable, V any](options Options) *Cache[K, V] {
	c := Cache[K, V]{
		de: options.defaultTTL,
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

func (c *Cache[K, V]) Close() {
	c.l.Lock() // deadlock on reentry.
	close(c.ch)
	c.ch = nil
}

func (o Options) SetDefaultTTL(d time.Duration) Options {
	o.defaultTTL = d
	return o
}
