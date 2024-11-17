package ttlcache

import "time"

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
				d = c.o.defaultTTL
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

		c.deleteUnsafe(k, v, ReasonTimedOut)
	}

	if !soon.IsZero() {
		c.ch <- soon.Sub(t)
	}
}
