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

// NoopExecutor is a no-op Executor used in round 1 and in tests. It logs the task
// and returns nil. It carries no state and is safe for concurrent use.
type NoopExecutor struct{}

func NewNoopExecutor() *NoopExecutor { return &NoopExecutor{} }

func (NoopExecutor) Execute(ctx context.Context, task *domain.Task) error {
	log.Info().
		Any("task_id", task.TaskId).
		Any("name", task.Name).
		Any("protocol", task.Protocol.String()).
		Any("address", task.Address).
		Msg("scheduler: noop execute")
	return nil
}
