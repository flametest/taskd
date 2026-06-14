package timingwheel

import (
	"reflect"
	"testing"
	"time"
)

func newTestTimer(key string) *timer {
	return &timer{key: key, expiration: time.Now(), fn: func() {}}
}

func TestBucket_PushAndLen(t *testing.T) {
	b := newBucket()
	if got := b.len(); got != 0 {
		t.Fatalf("len() = %d, want 0", got)
	}
	t1 := newTestTimer("a")
	t2 := newTestTimer("b")
	b.push(t1)
	b.push(t2)
	if got := b.len(); got != 2 {
		t.Errorf("len() = %d, want 2", got)
	}
	if t1.bucket != b || t2.bucket != b {
		t.Errorf("push did not set bucket back-pointer")
	}
	if t1.elem == nil || t2.elem == nil {
		t.Errorf("push did not set element back-pointer")
	}
}

func TestBucket_Remove(t *testing.T) {
	b := newBucket()
	t1 := newTestTimer("a")
	t2 := newTestTimer("b")
	b.push(t1)
	b.push(t2)
	b.remove(t1)
	if got := b.len(); got != 1 {
		t.Errorf("len() after remove = %d, want 1", got)
	}
	if t1.elem != nil || t1.bucket != nil {
		t.Errorf("remove did not clear timer back-pointers")
	}
	// the remaining timer is untouched
	if t2.elem == nil || t2.bucket != b {
		t.Errorf("remove corrupted the unrelated timer")
	}
}

func TestBucket_RemoveNotPresent(t *testing.T) {
	b := newBucket()
	t1 := newTestTimer("a")
	// never pushed; remove must be a no-op without panic
	b.remove(t1)
	if got := b.len(); got != 0 {
		t.Errorf("len() = %d, want 0", got)
	}
}

func TestBucket_Drain(t *testing.T) {
	b := newBucket()
	t1 := newTestTimer("a")
	t2 := newTestTimer("b")
	t3 := newTestTimer("c")
	b.push(t1)
	b.push(t2)
	b.push(t3)
	got := b.drain()
	want := []*timer{t1, t2, t3}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("drain() = %v, want %v", got, want)
	}
	if b.len() != 0 {
		t.Errorf("len() after drain = %d, want 0", b.len())
	}
	for _, tm := range got {
		if tm.elem != nil || tm.bucket != nil {
			t.Errorf("drain did not clear back-pointers for %q", tm.key)
		}
	}
}
