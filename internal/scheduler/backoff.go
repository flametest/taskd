package scheduler

import (
	"math/rand"
	"time"
)

// exponentialBackoff returns the retry delay before the next attempt, using
// exponential growth (base * 2^attempts) capped at max, with equal jitter: the
// result lies in [d/2, d) so retries never fire immediately. attempts is the
// number of failures already suffered (the value before it is incremented).
func exponentialBackoff(attempts int, base, max time.Duration) time.Duration {
	d := base
	for i := 0; i < attempts && d < max; i++ {
		d *= 2
	}
	// Cap at max; the second clause guards against int64 overflow.
	if d > max || d < base {
		d = max
	}
	half := d / 2
	if half <= 0 {
		return d
	}
	return half + time.Duration(rand.Int63n(int64(half)))
}
