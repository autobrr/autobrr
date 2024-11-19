package timecache

import (
	"testing"
	"time"
)

func TestTime(t *testing.T) {
	t.Parallel()
	tc := (&Cache{}).Now()
	if tc.IsZero() {
		t.Fatalf("time is zero")
	}
}

func TestRounding(t *testing.T) {
	t.Parallel()
	ti := New(Options{}.Round(time.Minute * 5)).Now()

	if ti.Minute()%5 != 0 {
		t.Fatalf("time is not a 5 multiple")
	}
}

func TestResolution(t *testing.T) {
	t.Parallel()
	const magicNumber = 3
	const rounds = 700
	ti := New(Options{}.Round(time.Millisecond * magicNumber))

	unique := 0
	old := ti.Now().UnixMilli()
	for i := 0; i < rounds; i++ {
		new := ti.Now().UnixMilli()
		if new > old {
			unique++
			old = new
		}

		if div := new % magicNumber; div != 0 {
			t.Fatalf("not a multiple of %d: %d", magicNumber, div)
		}
		time.Sleep(time.Millisecond * 1)
	}

	if unique < rounds/magicNumber-1 {
		t.Fatalf("not enough resolution rounds %d", unique)
	}
}
