package timingwheel

import "time"

// Clock abstracts the source of "now" so the wheel can be driven by a
// controllable clock in tests instead of wall time. Production uses
// systemClock; tests inject a fakeClock whose time advances only when the test
// calls advance.
type Clock interface {
	Now() time.Time
}

// systemClock reports the real wall-clock time.
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// fakeClock holds a manually controlled current time. It is not safe for
// concurrent use; tests drive it from a single goroutine via advance.
type fakeClock struct {
	now time.Time
}

func newFakeClock(start time.Time) *fakeClock {
	return &fakeClock{now: start}
}

func (f *fakeClock) Now() time.Time { return f.now }

// advance moves the fake clock forward by d. The caller is expected to pass a
// positive duration; the wheel only ever moves forward.
func (f *fakeClock) advance(d time.Duration) {
	f.now = f.now.Add(d)
}
