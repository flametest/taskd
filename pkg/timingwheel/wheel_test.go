package timingwheel

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// firedRecorder collects the keys of callbacks that have fired, via a buffered
// channel so recording from a dispatched goroutine never blocks.
type firedRecorder struct {
	ch chan string
}

func newFiredRecorder() *firedRecorder {
	return &firedRecorder{ch: make(chan string, 1024)}
}

func (r *firedRecorder) cb(key string) func() {
	return func() { r.ch <- key }
}

// count returns how many callbacks have fired so far (non-blocking).
func (r *firedRecorder) count() int { return len(r.ch) }

// drain returns and clears all fired keys in arrival order.
func (r *firedRecorder) drain() []string {
	out := make([]string, 0)
	for {
		select {
		case k := <-r.ch:
			out = append(out, k)
		default:
			return out
		}
	}
}

// waitRec polls until at least n callbacks have fired or the timeout elapses.
// Used for positive ("should fire") assertions: callbacks dispatch via
// go fn() so they run asynchronously after tick returns.
func waitRec(t *testing.T, rec *firedRecorder, n int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if rec.count() >= n {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected %d fires within %v, got %d", n, timeout, rec.count())
		}
		time.Sleep(time.Millisecond)
	}
}

// newTestWheel returns a manual-mode wheel (fake clock) that is already started.
// Tests advance logical time via advance.
func newTestWheel(t testing.TB) *hierarchicalWheel {
	t.Helper()
	w := New(
		WithTickInterval(10*time.Millisecond),
		WithSlotsPerLevel(8),
		WithMaxLevels(3),
		WithClock(newFakeClock(time.Unix(1_700_000_000, 0))),
	).(*hierarchicalWheel)
	w.Start()
	return w
}

// advance moves the fake clock forward by n ticks and runs each tick.
func (w *hierarchicalWheel) advance(n int) {
	fc := w.clock.(*fakeClock)
	for i := 0; i < n; i++ {
		fc.advance(w.tickInterval)
		w.tick()
	}
}

func TestAdd_FiresAfterDelay(t *testing.T) {
	w := newTestWheel(t)
	defer w.Stop()
	rec := newFiredRecorder()
	if err := w.Add(2*w.tickInterval, "a", rec.cb("a")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	w.advance(1)
	if rec.count() != 0 {
		t.Fatalf("fired too early: %d", rec.count())
	}
	w.advance(1)
	waitRec(t, rec, 1, 100*time.Millisecond)
}

func TestAdd_NonPositiveDelay_FiresImmediatelyAndNotCounted(t *testing.T) {
	for _, delay := range []time.Duration{0, -5 * time.Millisecond} {
		w := newTestWheel(t)
		rec := newFiredRecorder()
		if err := w.Add(delay, "a", rec.cb("a")); err != nil {
			t.Fatalf("Add(%v): %v", delay, err)
		}
		if s := w.Size(); s != 0 {
			t.Errorf("delay %v: Size = %d, want 0 (fire-and-forget not tracked)", delay, s)
		}
		waitRec(t, rec, 1, 100*time.Millisecond)
		w.Stop()
	}
}

func TestAdd_DuplicateKey_ReturnsErrKeyExists(t *testing.T) {
	w := newTestWheel(t)
	defer w.Stop()
	if err := w.Add(5*w.tickInterval, "a", func() {}); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	err := w.Add(5*w.tickInterval, "a", func() {})
	if !errors.Is(err, ErrKeyExists) {
		t.Fatalf("second Add error = %v, want ErrKeyExists", err)
	}
	if s := w.Size(); s != 1 {
		t.Errorf("Size after duplicate = %d, want 1", s)
	}
}

func TestAdd_AfterStop_ReturnsErrStopped(t *testing.T) {
	w := newTestWheel(t)
	w.Stop()
	err := w.Add(5*w.tickInterval, "a", func() {})
	if !errors.Is(err, ErrStopped) {
		t.Fatalf("Add after Stop error = %v, want ErrStopped", err)
	}
}

func TestAdd_BeforeStart_Panics(t *testing.T) {
	w := New(WithClock(newFakeClock(time.Unix(0, 0)))).(*hierarchicalWheel)
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on Add before Start")
		}
	}()
	_ = w.Add(5*time.Millisecond, "a", func() {})
}

func TestRemove_BeforeFire_Cancels(t *testing.T) {
	w := newTestWheel(t)
	defer w.Stop()
	rec := newFiredRecorder()
	_ = w.Add(3*w.tickInterval, "a", rec.cb("a"))
	if !w.Remove("a") {
		t.Fatalf("Remove returned false for existing key")
	}
	w.advance(5)
	if rec.count() != 0 {
		t.Fatalf("cancelled timer fired: %d", rec.count())
	}
	if s := w.Size(); s != 0 {
		t.Errorf("Size after remove = %d, want 0", s)
	}
}

func TestRemove_UnknownKey_ReturnsFalse(t *testing.T) {
	w := newTestWheel(t)
	defer w.Stop()
	if w.Remove("nope") {
		t.Fatalf("Remove of unknown key returned true")
	}
}

func TestRemove_AlreadyFired_ReturnsFalse(t *testing.T) {
	w := newTestWheel(t)
	defer w.Stop()
	rec := newFiredRecorder()
	_ = w.Add(1*w.tickInterval, "a", rec.cb("a"))
	w.advance(1)
	waitRec(t, rec, 1, 100*time.Millisecond)
	if w.Remove("a") {
		t.Fatalf("Remove of already-fired timer returned true")
	}
}

func TestSize_Accounting(t *testing.T) {
	w := newTestWheel(t)
	defer w.Stop()
	rec := newFiredRecorder()
	_ = w.Add(1*w.tickInterval, "a", rec.cb("a"))
	_ = w.Add(2*w.tickInterval, "b", rec.cb("b"))
	_ = w.Add(3*w.tickInterval, "c", rec.cb("c"))
	if s := w.Size(); s != 3 {
		t.Fatalf("Size = %d, want 3", s)
	}
	w.advance(1) // a fires; size updated synchronously inside tick
	if s := w.Size(); s != 2 {
		t.Errorf("after a fires, Size = %d, want 2", s)
	}
	w.advance(1) // b fires
	if s := w.Size(); s != 1 {
		t.Errorf("after b fires, Size = %d, want 1", s)
	}
	w.advance(1) // c fires
	if s := w.Size(); s != 0 {
		t.Errorf("after c fires, Size = %d, want 0", s)
	}
}

func TestCascade_FromHigherLevel(t *testing.T) {
	w := newTestWheel(t)
	defer w.Stop()
	rec := newFiredRecorder()
	// delay beyond level-0 span (8 ticks) sits in level 1, cascades down.
	_ = w.Add(10*w.tickInterval, "a", rec.cb("a"))
	w.advance(8) // one level-0 revolution -> cascade from level 1, not yet due
	if rec.count() != 0 {
		t.Fatalf("fired during cascade: %d", rec.count())
	}
	w.advance(2) // remaining 2 ticks -> fire
	waitRec(t, rec, 1, 100*time.Millisecond)
}

func TestCascade_DeepLevel(t *testing.T) {
	w := newTestWheel(t)
	defer w.Stop()
	rec := newFiredRecorder()
	// delay in level-2 range (> 64 ticks) cascades through two levels.
	_ = w.Add(70*w.tickInterval, "a", rec.cb("a"))
	w.advance(69)
	if rec.count() != 0 {
		t.Fatalf("fired before its time: %d", rec.count())
	}
	w.advance(1)
	waitRec(t, rec, 1, 100*time.Millisecond)
}

// TestFireOrder verifies each timer fires on its tick by advancing in steps and
// draining the new arrivals at each step (avoids relying on cross-goroutine
// callback ordering).
func TestFireOrder(t *testing.T) {
	w := newTestWheel(t)
	defer w.Stop()
	rec := newFiredRecorder()
	_ = w.Add(5*w.tickInterval, "a", rec.cb("a"))
	_ = w.Add(2*w.tickInterval, "b", rec.cb("b"))
	_ = w.Add(8*w.tickInterval, "c", rec.cb("c"))

	w.advance(2) // tick 2 -> b fires
	waitRec(t, rec, 1, 100*time.Millisecond)
	if got := fmt.Sprint(rec.drain()); got != "[b]" {
		t.Errorf("at tick 2 = %v, want [b]", got)
	}
	w.advance(3) // tick 5 -> a fires
	waitRec(t, rec, 1, 100*time.Millisecond)
	if got := fmt.Sprint(rec.drain()); got != "[a]" {
		t.Errorf("at tick 5 = %v, want [a]", got)
	}
	w.advance(3) // tick 8 -> c fires
	waitRec(t, rec, 1, 100*time.Millisecond)
	if got := fmt.Sprint(rec.drain()); got != "[c]" {
		t.Errorf("at tick 8 = %v, want [c]", got)
	}
}

func TestPrecisionBoundary(t *testing.T) {
	for k := 1; k <= 8; k++ {
		t.Run(fmt.Sprintf("k=%d", k), func(t *testing.T) {
			w := newTestWheel(t)
			defer w.Stop()
			rec := newFiredRecorder()
			if err := w.Add(time.Duration(k)*w.tickInterval, "a", rec.cb("a")); err != nil {
				t.Fatalf("Add: %v", err)
			}
			w.advance(k - 1)
			if rec.count() != 0 {
				t.Fatalf("fired %d ticks early (after %d/%d)", rec.count(), k-1, k)
			}
			w.advance(1)
			waitRec(t, rec, 1, 100*time.Millisecond)
		})
	}
}

func TestStart_Idempotent(t *testing.T) {
	w := New(WithClock(newFakeClock(time.Unix(0, 0)))).(*hierarchicalWheel)
	defer w.Stop()
	w.Start()
	w.Start() // must not panic or spawn a second driver
	rec := newFiredRecorder()
	_ = w.Add(2*w.tickInterval, "a", rec.cb("a"))
	w.advance(2)
	waitRec(t, rec, 1, 100*time.Millisecond)
}

func TestStop_Idempotent(t *testing.T) {
	w := newTestWheel(t)
	w.Stop()
	w.Stop() // must not panic (no close-of-closed-channel)
}

func TestStop_NoNewTimersFire(t *testing.T) {
	w := newTestWheel(t)
	rec := newFiredRecorder()
	_ = w.Add(3*w.tickInterval, "a", rec.cb("a"))
	_ = w.Add(4*w.tickInterval, "b", rec.cb("b"))
	w.Stop()
	w.advance(5)
	if rec.count() != 0 {
		t.Fatalf("timer fired after Stop: %d", rec.count())
	}
	if s := w.Size(); s != 0 {
		t.Errorf("Size after Stop = %d, want 0", s)
	}
}

func TestPickLevel(t *testing.T) {
	w := newTestWheel(t)
	defer w.Stop()
	cases := []struct {
		remainingTicks int
		wantLevel      int
	}{
		{0, 0}, {1, 0}, {7, 0},
		{8, 1}, {9, 1}, {63, 1},
		{64, 2}, {65, 2}, {511, 2},
	}
	for _, c := range cases {
		if got := w.pickLevel(c.remainingTicks); got != c.wantLevel {
			t.Errorf("pickLevel(%d) = %d, want %d", c.remainingTicks, got, c.wantLevel)
		}
	}
}

// TestConcurrency runs the production wheel (real ticker) under -race with many
// concurrent Add/Remove/Size callers interleaved with ticks.
func TestConcurrency(t *testing.T) {
	w := New(
		WithTickInterval(5*time.Millisecond),
		WithSlotsPerLevel(8),
		WithMaxLevels(3),
	).(*hierarchicalWheel)
	w.Start()
	defer w.Stop()

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 300; i++ {
				key := fmt.Sprintf("k-%d-%d", g, i)
				if err := w.Add(time.Duration(1+(i%10))*time.Millisecond, key, func() {}); err != nil {
					// collisions / stopped are fine under concurrency churn
					continue
				}
				if i%3 == 0 {
					w.Remove(key)
				}
				_ = w.Size()
			}
		}(g)
	}
	wg.Wait()
	if s := w.Size(); s < 0 {
		t.Fatalf("Size negative after concurrency: %d", s)
	}
}

// TestRealClock_SmokeTest is the single test that drives the wheel with real
// wall-clock time, confirming end-to-end firing latency.
func TestRealClock_SmokeTest(t *testing.T) {
	w := New(WithTickInterval(50 * time.Millisecond)).(*hierarchicalWheel)
	w.Start()
	defer w.Stop()

	rec := newFiredRecorder()
	start := time.Now()
	if err := w.Add(200*time.Millisecond, "a", rec.cb("a")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if rec.count() != 0 {
		t.Fatalf("fired too early: %d", rec.count())
	}
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && rec.count() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if rec.count() != 1 {
		t.Fatalf("expected 1 fire within 500ms, got %d", rec.count())
	}
	if elapsed := time.Since(start); elapsed < 150*time.Millisecond {
		t.Errorf("fired too early: %v", elapsed)
	}
}
