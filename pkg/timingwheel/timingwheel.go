package timingwheel

import "time"

// TimingWheel is a local timer scheduler.
//
// After [TimingWheel.Start] the instance runs a background goroutine that drives
// the wheel; the owner must call [TimingWheel.Stop] to release it when done.
// Calling Add after Stop returns [ErrStopped].
//
// Lifecycle contract: Start and Stop are not safe for concurrent use and must be
// driven serially by a single owner (e.g. the scheduler), whereas Add and Remove
// may be called concurrently from multiple goroutines.
type TimingWheel interface {
	// Start launches the background driver goroutine. Idempotent.
	Start()

	// Stop tears down the background goroutine. Timers that have not yet fired
	// will no longer be invoked. The caller should trigger Stop on ctx.Done.
	// Idempotent.
	Stop()

	// Add registers a timer that fires after delay; key is used later by
	// [TimingWheel.Remove]. When delay <= 0 the timer fires as soon as possible.
	// Returns [ErrKeyExists] if key is already present, or [ErrStopped] if the
	// wheel has been stopped.
	Add(delay time.Duration, key string, fn func()) error

	// Remove cancels a timer that has not yet fired, reporting whether key was
	// present and removed.
	Remove(key string) bool

	// Size returns the number of registered timers that have not yet fired,
	// for observability.
	Size() int
}
