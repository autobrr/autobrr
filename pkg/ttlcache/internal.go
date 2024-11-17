package ttlcache

import "time"

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
