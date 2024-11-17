package timecache

import (
	"sync"
	"time"
)

type Cache struct {
	m sync.RWMutex
	t time.Time
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

	t.t = time.Unix(time.Now().Unix(), 0)

	go func() {
		time.Sleep(1 * time.Second)
		t.reset()
	}()

	return t.t
}

func (t *Cache) reset() {
	t.m.Lock()
	defer t.m.Unlock()
	t.t = time.Time{}
}
