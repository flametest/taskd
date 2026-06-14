// Package timingwheel implements a hierarchical timing wheel for efficiently
// scheduling a large number of short-lived timers within a single process.
//
// Design goals:
//   - O(1) amortized Add / Remove / tick
//   - Concurrent registration and cancellation
//   - Trigger precision bounded by the system clock and the ticker interval,
//     typically reaching millisecond granularity
//
// Role within taskd:
// After the scheduler claims an about-to-due task from the database, it computes
// the delay as (exec_time - now) and registers the task on a local TimingWheel
// via [TimingWheel.Add]. When the wheel fires, the callback hands the task off
// to a worker for execution. This package is unaware of any taskd business
// semantics; it is organized purely around delay / key / callback, which keeps
// it independently unit-testable and easy to extract into its own repository or
// fold into the vita base library later.
//
// Triggering contract:
// Callbacks are invoked on a goroutine distinct from the caller's. Implementations
// must ensure that a single blocking callback does not stall the firing of other
// timers; callbacks should therefore return quickly or defer their own work
// asynchronously.
package timingwheel
