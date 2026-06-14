package timingwheel

import (
	"container/list"
	"time"
)

// timer is one scheduled callback. While waiting to fire it lives in exactly
// one bucket of one level. Removal is O(1) because each timer keeps a
// back-pointer to its bucket and the *list.Element it occupies within that
// bucket's list.
type timer struct {
	key        string
	expiration time.Time
	fn         func()
	bucket     *bucket
	elem       *list.Element
}

// bucket is one slot in one level of the wheel: a doubly-linked list of timers
// that share the same (level, slot) position at insertion time.
type bucket struct {
	list *list.List
}

func newBucket() *bucket {
	return &bucket{list: list.New()}
}

// push appends t to the bucket and records the element and bucket back-pointers
// on t.
func (b *bucket) push(t *timer) {
	t.bucket = b
	t.elem = b.list.PushBack(t)
}

// remove detaches t from its bucket. It is O(1) via the stored element and is a
// no-op when t is not currently in a bucket.
func (b *bucket) remove(t *timer) {
	if t.elem == nil {
		return
	}
	b.list.Remove(t.elem)
	t.elem = nil
	t.bucket = nil
}

// drain pops every timer from the bucket in insertion order and clears each
// timer's element/bucket back-pointers. The bucket is left empty.
func (b *bucket) drain() []*timer {
	out := make([]*timer, 0, b.list.Len())
	for {
		e := b.list.Front()
		if e == nil {
			break
		}
		t := e.Value.(*timer)
		b.list.Remove(e)
		t.elem = nil
		t.bucket = nil
		out = append(out, t)
	}
	return out
}

func (b *bucket) len() int {
	return b.list.Len()
}
