package scheduler

import (
	"context"

	"github.com/flametest/taskd/internal/domain"
	log "github.com/flametest/vita/vlog"
)

// Executor runs a claimed task. Round 1 ships only NoopExecutor; later rounds add
// HTTP/gRPC executors. Implementations must be safe for concurrent use by the
// worker pool.
type Executor interface {
	// Execute performs the task. A nil error means success; the scheduler then
	// MarkSucceeded. A non-nil error leaves the task in 'claimed' state (the
	// failure/retry path is a later round).
	Execute(ctx context.Context, task *domain.Task) error
}

// NonRetryableError wraps an error whose cause will not change by retrying (e.g.
// HTTP 4xx). The scheduler marks such tasks dead instead of retrying.
type NonRetryableError struct{ Err error }

func (e *NonRetryableError) Error() string { return e.Err.Error() }
func (e *NonRetryableError) Unwrap() error { return e.Err }

// NewNonRetryableError wraps err as a non-retryable error.
func NewNonRetryableError(err error) error { return &NonRetryableError{Err: err} }

// NoopExecutor is a no-op Executor used in round 1 and in tests. It logs the task
// and returns nil. It carries no state and is safe for concurrent use.
type NoopExecutor struct{}

func NewNoopExecutor() *NoopExecutor { return &NoopExecutor{} }

func (NoopExecutor) Execute(ctx context.Context, task *domain.Task) error {
	log.Info().
		Any("task_id", task.Id).
		Any("name", task.Name).
		Any("protocol", task.Protocol.String()).
		Any("address", task.Address).
		Msg("scheduler: noop execute")
	return nil
}
