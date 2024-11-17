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
