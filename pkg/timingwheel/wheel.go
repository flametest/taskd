package timingwheel

import (
	"sync"
	"time"
)

// Option configures a TimingWheel constructed by New.
type Option func(*hierarchicalWheel)

// WithTickInterval sets the duration of one wheel tick (default 100ms). A
// smaller interval gives finer trigger precision at higher CPU cost. Must be
// positive.
func WithTickInterval(d time.Duration) Option {
	return func(w *hierarchicalWheel) { w.tickInterval = d }
}

// WithSlotsPerLevel sets the number of slots per level (default 256). Larger
// values reduce cascade frequency at the cost of memory. Must be >= 2.
func WithSlotsPerLevel(n int) Option {
	return func(w *hierarchicalWheel) { w.slotsPerLevel = n }
}

// WithClock injects the clock used as the source of "now"; the default is the
// real wall clock. Passing a *fakeClock switches the wheel into manual mode:
// Start does not spawn a ticker goroutine and tests drive ticks explicitly via
// the unexported advance helper.
func WithClock(c Clock) Option {
	return func(w *hierarchicalWheel) { w.clock = c }
}

// WithMaxLevels sets the number of levels in the hierarchy (default 4). With
// the default tick and slots, 4 levels span ~11 days. Must be >= 1.
func WithMaxLevels(n int) Option {
	return func(w *hierarchicalWheel) { w.numLevels = n }
}

// New constructs a TimingWheel with the given options. It panics on invalid
// option values (programmer error).
func New(opts ...Option) TimingWheel {
	w := &hierarchicalWheel{
		tickInterval:  100 * time.Millisecond,
		slotsPerLevel: 256,
		numLevels:     4,
		clock:         systemClock{},
		index:         make(map[string]*timer),
	}
	for _, o := range opts {
		o(w)
	}
	if w.tickInterval <= 0 {
		panic("timingwheel: tickInterval must be positive")
	}
	if w.slotsPerLevel < 2 {
		panic("timingwheel: slotsPerLevel must be >= 2")
	}
	if w.numLevels < 1 {
		panic("timingwheel: numLevels must be >= 1")
	}
	if w.clock == nil {
		w.clock = systemClock{}
	}
	if _, ok := w.clock.(*fakeClock); ok {
		w.manual = true
	}
	w.levels = make([][]*bucket, w.numLevels)
	for l := 0; l < w.numLevels; l++ {
		w.levels[l] = make([]*bucket, w.slotsPerLevel)
		for s := 0; s < w.slotsPerLevel; s++ {
			w.levels[l][s] = newBucket()
		}
	}
	return w
}

// hierarchicalWheel is a Kafka-style multi-level timing wheel. A timer whose
// delay fits in level 0 lives there; longer delays occupy coarser higher levels
// and cascade down as the wheel advances.
type hierarchicalWheel struct {
	clock         Clock
	tickInterval  time.Duration
	slotsPerLevel int
	numLevels     int

	mu          sync.Mutex
	levels      [][]*bucket
	index       map[string]*timer
	size        int
	currentTime time.Duration // monotonic logical time elapsed since Start
	startedAt   time.Time
	currentSlot []int

	manual  bool
	started bool
	stopped bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// Start launches the wheel. In production it spawns a ticker goroutine that
// calls tick on every interval; in manual mode (fake clock) it only records the
// start anchor and the caller drives ticks. Idempotent.
func (w *hierarchicalWheel) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.started || w.stopped {
		return
	}
	w.started = true
	w.startedAt = w.clock.Now()
	w.currentTime = 0
	w.currentSlot = make([]int, w.numLevels)
	if !w.manual {
		w.stopCh = make(chan struct{})
		w.doneCh = make(chan struct{})
		go w.loop()
	}
}

// Stop shuts the wheel down. No new timers fire after Stop returns; timers
// already dispatched (callback goroutine launched) may still complete. A
// stopped wheel cannot be restarted. Idempotent and safe to call before Start.
func (w *hierarchicalWheel) Stop() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	w.index = make(map[string]*timer)
	w.size = 0
	started := w.started
	manual := w.manual
	w.mu.Unlock()

	if started && !manual {
		close(w.stopCh)
		<-w.doneCh
	}
}

// Add registers a callback that fires after delay, identified by key. A
// non-positive delay fires the callback as soon as possible (fire-and-forget:
// not tracked, not cancelable, not counted by Size). Returns ErrKeyExists if
// key is already in use, or ErrStopped if the wheel has been stopped. Add must
// be called after Start.
func (w *hierarchicalWheel) Add(delay time.Duration, key string, fn func()) error {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return ErrStopped
	}
	if !w.started {
		w.mu.Unlock()
		panic("timingwheel: Add called before Start")
	}
	if _, ok := w.index[key]; ok {
		w.mu.Unlock()
		return ErrKeyExists
	}
	if delay <= 0 {
		w.mu.Unlock()
		go fn()
		return nil
	}
	t := &timer{
		key:        key,
		expiration: w.clock.Now().Add(delay),
		fn:         fn,
	}
	w.place(t, int(t.expiration.Sub(w.logicalNow())/w.tickInterval))
	w.index[key] = t
	w.size++
	w.mu.Unlock()
	return nil
}

// Remove cancels the timer identified by key. Returns true if the timer was
// present and removed. A timer whose callback was already dispatched this tick
// may still fire; Remove only cancels future dispatch.
func (w *hierarchicalWheel) Remove(key string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	t, ok := w.index[key]
	if !ok {
		return false
	}
	if t.bucket != nil {
		t.bucket.remove(t)
	}
	delete(w.index, key)
	w.size--
	return true
}

// Size returns the number of registered timers that have not yet fired.
func (w *hierarchicalWheel) Size() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.size
}

// loop is the production tick driver: it advances the wheel on each ticker
// firing until Stop closes stopCh.
func (w *hierarchicalWheel) loop() {
	ticker := time.NewTicker(w.tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			w.tick()
		case <-w.stopCh:
			close(w.doneCh)
			return
		}
	}
}

// tick advances the wheel by exactly one tickInterval: it drains the next
// level-0 slot, cascades coarser-level slots as finer levels wrap, removes
// fired timers from the index, then dispatches their callbacks off the lock so
// user code can never stall the wheel.
func (w *hierarchicalWheel) tick() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.currentTime += w.tickInterval
	w.currentSlot[0] = (w.currentSlot[0] + 1) % w.slotsPerLevel

	fired := w.levels[0][w.currentSlot[0]].drain()

	level := 0
	for level+1 < w.numLevels && w.currentSlot[level] == 0 {
		level++
		w.currentSlot[level] = (w.currentSlot[level] + 1) % w.slotsPerLevel
		for _, t := range w.levels[level][w.currentSlot[level]].drain() {
			w.reinsert(t, &fired)
		}
	}

	for _, t := range fired {
		delete(w.index, t.key)
		w.size--
	}
	w.mu.Unlock()

	for _, t := range fired {
		go t.fn()
	}
}

// place inserts timer t into the bucket chosen for remainingTicks. The caller
// must hold w.mu. remainingTicks is clamped to 0; a non-positive value lands t
// in the current level-0 slot (it fires on the next revolution).
func (w *hierarchicalWheel) place(t *timer, remainingTicks int) {
	if remainingTicks < 0 {
		remainingTicks = 0
	}
	level := w.pickLevel(remainingTicks)
	slot := (w.currentSlot[level] + remainingTicks/w.levelBase(level)) % w.slotsPerLevel
	w.levels[level][slot].push(t)
}

// reinsert re-evaluates a cascaded timer against the current logical time and
// either parks it in a finer-level bucket or marks it due. The caller must hold
// w.mu; the timer must already be detached (drain cleared its bucket/element).
func (w *hierarchicalWheel) reinsert(t *timer, fired *[]*timer) {
	remaining := t.expiration.Sub(w.logicalNow())
	if remaining <= 0 {
		*fired = append(*fired, t)
		return
	}
	w.place(t, int(remaining/w.tickInterval))
}

// pickLevel returns the finest level that fully contains remainingTicks: the
// smallest L such that remainingTicks < slotsPerLevel^(L+1).
func (w *hierarchicalWheel) pickLevel(remainingTicks int) int {
	span := w.slotsPerLevel
	level := 0
	for level+1 < w.numLevels && remainingTicks >= span {
		span *= w.slotsPerLevel
		level++
	}
	return level
}

// levelBase returns slotsPerLevel^level.
func (w *hierarchicalWheel) levelBase(level int) int {
	base := 1
	for i := 0; i < level; i++ {
		base *= w.slotsPerLevel
	}
	return base
}

// logicalNow returns the wheel's monotonic current time: startedAt plus the
// elapsed tick duration. The caller must hold w.mu.
func (w *hierarchicalWheel) logicalNow() time.Time {
	return w.startedAt.Add(w.currentTime)
}
