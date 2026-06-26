package scheduler

import (
	"testing"
	"time"
)

// upperBound computes the pre-jitter cap (base * 2^attempts, capped at max).
func upperBound(attempts int, base, max time.Duration) time.Duration {
	d := base
	for i := 0; i < attempts && d < max; i++ {
		d *= 2
	}
	if d > max {
		d = max
	}
	return d
}

func TestExponentialBackoff_InJitterRange(t *testing.T) {
	base := 1 * time.Second
	max := 10 * time.Second
	for attempts := 0; attempts < 15; attempts++ {
		ub := upperBound(attempts, base, max)
		// run a few times since jitter is random
		for k := 0; k < 20; k++ {
			got := exponentialBackoff(attempts, base, max)
			if got < ub/2 || got >= ub {
				t.Errorf("attempts=%d got=%v not in [%v,%v)", attempts, got, ub/2, ub)
			}
		}
	}
}

func TestExponentialBackoff_CapsAtMax(t *testing.T) {
	base := 1 * time.Second
	max := 4 * time.Second
	for k := 0; k < 20; k++ {
		got := exponentialBackoff(100, base, max)
		// capped so jitter range is [max/2, max)
		if got < max/2 || got >= max {
			t.Errorf("got %v not in [%v,%v) (cap failed)", got, max/2, max)
		}
	}
}

func TestExponentialBackoff_NonZero(t *testing.T) {
	got := exponentialBackoff(0, 1*time.Second, 10*time.Second)
	if got <= 0 {
		t.Errorf("got non-positive %v", got)
	}
}
