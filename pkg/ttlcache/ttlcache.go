package ttlcache

import (
	"time"

	"github.com/titlerr/upgraderr/pkg/timecache"
)

func New[K comparable, V any](options Options[K, V]) *Cache[K, V] {
	c := Cache[K, V]{
		o:  options,
		ch: make(chan time.Time, 1000),
		m:  make(map[K]Item[V]),
	}

	if options.defaultTTL != NoTTL && options.defaultResolution == 0 {
		c.tc = *timecache.New(timecache.Options{}.Round(options.defaultTTL / 2))
	} else if options.defaultResolution != 0 {
		c.tc = *timecache.New(timecache.Options{}.Round(options.defaultResolution))
	}

	go c.startExpirations()
	return &c
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	it, ok := c.GetItem(key)
	if !ok {
		return *new(V), ok
	}

	return it.GetValue(), ok
}

func (c *Cache[K, V]) GetItem(key K) (Item[V], bool) {
	it, ok := c.get(key)
	if !ok {
		return it, ok
	}

	if !c.o.noUpdateTime && !it.t.IsZero() {
		if _, t := c.getDuration(it.d); t.After(it.t) {
			c.set(key, it)
		}
	}

	return it, ok
}

func (c *Cache[K, V]) GetOrSet(key K, value V, duration time.Duration) (V, bool) {
	it, ok := c.GetOrSetItem(key, value, duration)
	if !ok {
		return *new(V), ok
	}

	return it.GetValue(), ok
}

func (c *Cache[K, V]) fixupDuration(duration time.Duration) time.Duration {
	if c.o.defaultTTL == NoTTL && duration == DefaultTTL {
		return NoTTL
	}

	return duration
}

func (c *Cache[K, V]) GetOrSetItem(key K, value V, duration time.Duration) (Item[V], bool) {
	it, ok := c.getOrSet(key, Item[V]{v: value, d: c.fixupDuration(duration)})
	if !ok {
		return Item[V]{}, ok
	}

	return it, ok
}

func (c *Cache[K, V]) Set(key K, value V, duration time.Duration) bool {
	c.SetItem(key, value, duration)
	return true
}

func (c *Cache[K, V]) SetItem(key K, value V, duration time.Duration) Item[V] {
	return c.set(key, Item[V]{v: value, d: c.fixupDuration(duration)})
}

func (c *Cache[K, V]) Delete(key K) {
	c.delete(key, ReasonDeleted)
}

func (c *Cache[K, V]) GetKeys() []K {
	return c.getkeys()
}

func (c *Cache[K, V]) Close() {
	c.close()
}

func (i *Item[V]) GetDuration() time.Duration {
	return i.getDuration()
}

func (i *Item[V]) GetTime() time.Time {
	return i.getTime()
}

func (i *Item[V]) GetValue() V {
	return i.getValue()
}

func (o Options[K, V]) SetTimerResolution(d time.Duration) Options[K, V] {
	o.defaultResolution = d
	return o
}

func (o Options[K, V]) SetDefaultTTL(d time.Duration) Options[K, V] {
	o.defaultTTL = d
	return o
}

func (o Options[K, V]) SetDeallocationFunc(f DeallocationFunc[K, V]) Options[K, V] {
	o.deallocationFunc = f
	return o
}

func (o Options[K, V]) DisableUpdateTime(val bool) Options[K, V] {
	o.noUpdateTime = val
	return o
}
