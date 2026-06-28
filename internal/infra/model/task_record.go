package model

import (
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"gorm.io/datatypes"
)

// RecordBase is the append-only base for audit rows: a UUID primary key plus a
// created_at timestamp. Unlike vgorm.BasePostgres it carries no version (rows are
// immutable), no updated_at, and no soft-delete column — an audit log is never
// updated or deleted.
type RecordBase struct {
	Id        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CreatedAt time.Time `gorm:"<-:create;index;type:TIMESTAMPTZ;default:CURRENT_TIMESTAMP;not null;column:created_at"`
}

// TaskRecord is one row of the execution-audit log: the outcome of a single
// executor.Execute call (success, retryable failure, or non-retryable failure).
// It is append-only and intentionally NOT coupled to the task state machine:
// recording is best-effort and a failure to record never affects scheduling.
// task : task_record = 1 : N.
type TaskRecord struct {
	RecordBase
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
