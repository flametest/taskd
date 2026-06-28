package model

import (
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/vita/vgorm"
	"gorm.io/datatypes"
)

// TaskRecord is one row of the execution-audit log: the outcome of a single
// executor.Execute call (success, retryable failure, or non-retryable failure).
// It is append-only and intentionally NOT coupled to the task state machine:
// recording is best-effort and a failure to record never affects scheduling.
// task : task_record = 1 : N. It embeds vgorm.BasePostgres to stay consistent
// with model.Task; the version/updated_at/deleted_at columns are inert for an
// append-only row but harmless.
type TaskRecord struct {
	vgorm.BasePostgres
	TaskId       string        `gorm:"column:task_id"`
	Attempt      int           `gorm:"column:attempt"`
	Result       enum.Result   `gorm:"column:result"`
	Protocol     enum.Protocol `gorm:"column:protocol"`
	InstanceId   string        `gorm:"column:instance_id"`
	ErrorMessage string        `gorm:"column:error_message"`
	StartedAt    time.Time     `gorm:"column:started_at"`
	FinishedAt   time.Time     `gorm:"column:finished_at"`
	DurationMs   int64         `gorm:"column:duration_ms"`
	// Response is reserved for a future round that captures the upstream response
	// payload. It is left NULL this round.
	Response datatypes.JSON `gorm:"column:response"`
}
