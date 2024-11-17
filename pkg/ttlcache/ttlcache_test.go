package ttlcache

import (
	"fmt"
	"testing"
	"time"
	"unsafe"
)

func TestGet(t *testing.T) {
	c := New[int, bool](1 * time.Second)
	defer c.Close()

	fmt.Printf("sizeof Cache: %d\n", unsafe.Sizeof(c))
	for i := 0; i < 10; i++ {
		c.Set(i, true, DefaultTTL)
	}

	for i := 0; i < 10; i++ {
		val, ok := c.Get(i)
		if !ok {
			t.Fatalf("Missing key: %d", i)
		} else if !val {
			t.Fatalf("Bad value on key: %d", i)
		}
	}
}

func TestExpirations(t *testing.T) {
	c := New[int, bool](1 * time.Second)
	defer c.Close()
	for i := 0; i < 10; i++ {
		c.Set(i, true, DefaultTTL)
	}

	time.Sleep(3 * time.Second)

	for i := 0; i < 10; i++ {
		if _, ok := c.Get(i); ok {
			t.Fatalf("Found key: %d", i)
		}
	}
}

func TestSwaps(t *testing.T) {
	c := New[int, bool](1 * time.Second)
	defer c.Close()
	for i := 0; i < 10; i++ {
		c.Set(i, true, DefaultTTL)
	}

	time.Sleep(3 * time.Second)
	for i := 0; i < 10; i++ {
		if _, ok := c.Get(i); ok {
			t.Fatalf("Found key: %d", i)
		}
	}

	for i := 10; i < 20; i++ {
		c.Set(i, true, DefaultTTL)
		if _, ok := c.Get(i); !ok {
			t.Fatalf("Missing key: %d", i)
		}
	}
}

func TestReschedule(t *testing.T) {
	c := New[int, bool](1 * time.Second)
	defer c.Close()
	for i := 1; i < 10; i++ {
		c.Set(i, true, time.Duration(10-i)*time.Second)
	}

	time.Sleep(15 * time.Second)
	for i := 1; i < 10; i++ {
		if _, ok := c.Get(i); ok {
			t.Fatalf("Found key: %d", i)
		}
	}
}

func TestSchedule(t *testing.T) {
	c := New[int, bool](1 * time.Second)
	defer c.Close()
	for i := 1; i < 10; i++ {
		c.Set(i, true, time.Duration(i)*time.Second)
	}

	time.Sleep(15 * time.Second)
	for i := 1; i < 10; i++ {
		if _, ok := c.Get(i); ok {
			t.Fatalf("Found key: %d", i)
		}
	}
}

func TestInterlace(t *testing.T) {
	c := New[int, bool](1 * time.Second)
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

	time.Sleep(5 * time.Second)
	swap = false
	for i := 0; i < 10; i++ {
		swap = !swap
		if !swap {
			continue
		}

		if _, ok := c.Get(i); !ok {
			t.Fatalf("Found key: %d", i)
		}
	}
}
