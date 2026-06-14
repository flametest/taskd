package timingwheel

import (
	"reflect"
	"testing"
	"time"
)

func TestFakeClock_InitialNow(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	if got := fc.Now(); !reflect.DeepEqual(got, start) {
		t.Errorf("Now() = %v, want %v", got, start)
	}
}

func TestFakeClock_Advance(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	fc.advance(1500 * time.Millisecond)
	want := start.Add(1500 * time.Millisecond)
	if got := fc.Now(); !reflect.DeepEqual(got, want) {
		t.Errorf("Now() = %v, want %v", got, want)
	}
}

func TestFakeClock_AdvanceAccumulates(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	fc := newFakeClock(start)
	fc.advance(1 * time.Second)
	fc.advance(500 * time.Millisecond)
	want := start.Add(1500 * time.Millisecond)
	if got := fc.Now(); !reflect.DeepEqual(got, want) {
		t.Errorf("Now() = %v, want %v", got, want)
	}
}
