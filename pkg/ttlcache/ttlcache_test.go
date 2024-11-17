package ttlcache

import (
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	c := New[int, bool](Options[int, bool]{}.SetDefaultTTL(1 * time.Second))
	defer c.Close()

	for i := 0; i < 10; i++ {
		c.Set(i, true, DefaultTTL)
	}

	for i := 0; i < 10; i++ {
		val, ok := c.Get(i)
		if !ok {
			t.Fatalf("missing key: %d", i)
		} else if !val {
			t.Fatalf("bad value on key: %d", i)
		}
	}
}

func TestExpirations(t *testing.T) {
	c := New[int, bool](Options[int, bool]{}.SetDefaultTTL(200 * time.Millisecond))
	defer c.Close()
	for i := 0; i < 10; i++ {
		c.Set(i, true, DefaultTTL)
	}

	time.Sleep(3 * time.Second)

	for i := 0; i < 10; i++ {
		if _, ok := c.Get(i); ok {
			t.Fatalf("found key: %d", i)
		}
	}
}

func TestSwaps(t *testing.T) {
	c := New[int, bool](Options[int, bool]{}.SetDefaultTTL(200 * time.Millisecond))
	defer c.Close()
	for i := 0; i < 10; i++ {
		c.Set(i, true, DefaultTTL)
	}

	time.Sleep(5 * time.Second)
	for i := 0; i < 10; i++ {
		if _, ok := c.Get(i); ok {
			t.Fatalf("found key: %d", i)
		}
	}

	for i := 10; i < 20; i++ {
		c.Set(i, true, DefaultTTL)
		if _, ok := c.Get(i); !ok {
			t.Fatalf("missing key: %d", i)
		}
	}
}

func TestRetimer(t *testing.T) {
	c := New[int, bool](Options[int, bool]{}.SetDefaultTTL(200 * time.Millisecond))
	defer c.Close()
	for i := 1; i < 10; i++ {
		c.Set(i, true, time.Duration(10-i)*100*time.Millisecond)
	}

	time.Sleep(5 * time.Second)
	for i := 1; i < 10; i++ {
		if _, ok := c.Get(i); ok {
			t.Fatalf("found key: %d", i)
		}
	}
}

func TestSchedule(t *testing.T) {
	c := New[int, bool](Options[int, bool]{}.SetDefaultTTL(1 * time.Second))
	defer c.Close()
	for i := 1; i < 10; i++ {
		c.Set(i, true, time.Duration(i)*100*time.Millisecond)
	}

	time.Sleep(3 * time.Second)
	for i := 1; i < 10; i++ {
		if _, ok := c.Get(i); ok {
			t.Fatalf("found key: %d", i)
		}
	}
}

func TestInterlace(t *testing.T) {
	c := New[int, bool](Options[int, bool]{}.SetDefaultTTL(1 * time.Second))
	defer c.Close()
	swap := false
	for i := 0; i < 10; i++ {
		swap = !swap
		ttl := DefaultTTL
		if swap {
			ttl = NoTTL
		}
		c.Set(i, true, ttl)
	}

	time.Sleep(3 * time.Second)
	swap = false
	for i := 0; i < 10; i++ {
		swap = !swap
		if !swap {
			continue
		}

		if _, ok := c.Get(i); !ok {
			t.Fatalf("found key: %d", i)
		}
	}
}

func TestReschedule(t *testing.T) {
	c := New[int, bool](Options[int, bool]{}.SetDefaultTTL(1 * time.Second))
	defer c.Close()
	for i := 1; i < 10; i++ {
		c.Set(i, true, NoTTL)
		c.Set(i, true, DefaultTTL)
	}

	time.Sleep(3 * time.Second)
	for i := 1; i < 10; i++ {
		if _, ok := c.Get(i); ok {
			t.Fatalf("found key: %d", i)
		}
	}
}

func TestDelete(t *testing.T) {
	c := New[int, bool](Options[int, bool]{}.SetDefaultTTL(1 * time.Second))
	defer c.Close()
	for i := 1; i < 10; i++ {
		c.Set(i, true, NoTTL)
		c.Delete(i)
	}

	for i := 1; i < 10; i++ {
		if _, ok := c.Get(i); ok {
			t.Fatalf("found key: %d", i)
		}
	}
}

func TestDeallocationTimeout(t *testing.T) {
	hit := false
	o := Options[int, bool]{}.
		SetDefaultTTL(time.Millisecond * 100).
		SetDeallocationFunc(func(key int, value bool, reason DeallocationReason) { hit = reason == ReasonTimedOut })

	c := New[int, bool](o)
	defer c.Close()

	for i := 0; i < 1; i++ {
		c.Set(i, true, DefaultTTL)
	}

	time.Sleep(3 * time.Second)
	if !hit {
		t.Fatalf("Deallocation not hit.")
	}
}

func TestDeallocationDeleted(t *testing.T) {
	hit := false
	o := Options[int, bool]{}.
		SetDefaultTTL(time.Millisecond * 100).
		SetDeallocationFunc(func(key int, value bool, reason DeallocationReason) { hit = reason == ReasonDeleted })

	c := New[int, bool](o)
	defer c.Close()

	for i := 0; i < 1; i++ {
		c.Set(i, true, DefaultTTL)
		c.Delete(i)
	}

	if !hit {
		t.Fatalf("Deallocation not hit.")
	}
}
