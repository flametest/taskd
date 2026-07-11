package scheduler

import (
	"context"

	"github.com/flametest/taskd/internal/domain"
	log "github.com/flametest/vita/vlog"
)

// ExecutionResponse captures the upstream response of one execution for the audit
// log (stored in task_record.response). Status is a protocol-specific status
// (HTTP status code string, or gRPC code string); Body is a truncated textual
// representation of the response payload. Both fields are best-effort and may be
// empty (e.g. gRPC's RunResponse carries no business payload).
type ExecutionResponse struct {
	Status string `json:"status,omitempty"`
	Body   string `json:"body,omitempty"`
}

// Executor runs a claimed task. Implementations must be safe for concurrent use
// by the worker pool. Execute returns the upstream response (for auditing; may
// be nil when no response was received, e.g. a connect failure or a panic) and
// an error (nil on success).
type Executor interface {
	Execute(ctx context.Context, task *domain.Task) (*ExecutionResponse, error)
}

// NonRetryableError wraps an error whose cause will not change by retrying (e.g.
// HTTP 4xx). The scheduler marks such tasks dead instead of retrying.
type NonRetryableError struct{ Err error }

func (e *NonRetryableError) Error() string { return e.Err.Error() }
func (e *NonRetryableError) Unwrap() error { return e.Err }

// NewNonRetryableError wraps err as a non-retryable error.
func NewNonRetryableError(err error) error { return &NonRetryableError{Err: err} }

// NoopExecutor is a no-op Executor used in tests. It logs the task and returns
// (nil, nil). It carries no state and is safe for concurrent use.
type NoopExecutor struct{}

func NewNoopExecutor() *NoopExecutor { return &NoopExecutor{} }

func (NoopExecutor) Execute(ctx context.Context, task *domain.Task) (*ExecutionResponse, error) {
	log.Info().
		Any("task_id", task.Id).
		Any("name", task.Name).
		Any("protocol", task.Protocol.String()).
		Any("address", task.Address).
		Msg("scheduler: noop execute")
	return nil, nil
}
