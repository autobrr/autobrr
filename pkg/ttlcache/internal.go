package ttlcache

import "time"

func (c *Cache[K, V]) get(key K) (Item[V], bool) {
	c.l.RLock()
	defer c.l.RUnlock()
	return c._g(key)
}

func (c *Cache[K, V]) _g(key K) (Item[V], bool) {
	v, ok := c.m[key]
	if !ok {
		return v, ok
	}

	return v, ok
}

func (c *Cache[K, V]) set(key K, it Item[V]) Item[V] {
	c.l.Lock()
	defer c.l.Unlock()
	return c._s(key, it)
}

func (c *Cache[K, V]) _s(key K, it Item[V]) Item[V] {
	it.d, it.t = c.getDuration(it.d)
	c.m[key] = it
	c.ch <- it.t
	return it
}

func (c *Cache[K, V]) getOrSet(key K, it Item[V]) (Item[V], bool) {
	c.l.Lock()
	defer c.l.Unlock()
	return c._gos(key, it)
}

func (c *Cache[K, V]) _gos(key K, it Item[V]) (Item[V], bool) {
	if g, ok := c._g(key); ok {
		return g, ok
	}

	return c._s(key, it), true
}

func (c *Cache[K, V]) delete(key K, reason DeallocationReason) {
	var v Item[V]
	c.l.Lock()
	defer c.l.Unlock()

	if c.o.deallocationFunc != nil {
		var ok bool
		v, ok = c.m[key]
		if !ok {
			return
		}
	}

	c.deleteUnsafe(key, v, reason)
}

func (c *Cache[K, V]) deleteUnsafe(key K, v Item[V], reason DeallocationReason) {
	delete(c.m, key)

	if c.o.deallocationFunc != nil {
		c.o.deallocationFunc(key, v.v, reason)
	}
}

func (c *Cache[K, V]) getkeys() []K {
	c.l.RLock()
	defer c.l.RUnlock()

	keys := make([]K, len(c.m))
	for k := range c.m {
		keys = append(keys, k)
	}

	return keys
}

func (c *Cache[K, V]) close() {
	c.l.Lock()
	defer c.l.Unlock()
	close(c.ch)
}

func (c *Cache[K, V]) getDuration(d time.Duration) (time.Duration, time.Time) {
	switch d {
	case NoTTL:
	case DefaultTTL:
		return c.o.defaultTTL, c.tc.Now().Add(c.o.defaultTTL)
	default:
		return d, c.tc.Now().Add(d)
	}

	return NoTTL, time.Time{}
}

func (i *Item[V]) getDuration() time.Duration {
	return i.d
}

func (i *Item[V]) getTime() time.Time {
	return i.t
}

func (i *Item[V]) getValue() V {
	return i.v
}
