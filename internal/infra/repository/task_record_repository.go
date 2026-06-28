package repository

import (
	"context"

	"github.com/flametest/taskd/internal/infra/model"
	"gorm.io/gorm"
)

// TaskRecordRepository appends and queries execution-audit rows.
type TaskRecordRepository interface {
	// Record appends one execution-audit row. Best-effort by contract: callers
	// must log and swallow any error so auditing never affects the task state
	// machine.
	Record(ctx context.Context, r *model.TaskRecord) error
	// ListByTaskId returns up to limit records for taskId, newest-first. limit is
	// clamped (<=0 or >maxListLimit -> defaultListLimit); offset is the zero-based
	// skip for pagination and clamped to >=0.
	ListByTaskId(ctx context.Context, taskId string, limit, offset int) ([]*model.TaskRecord, error)
}

const (
	defaultListLimit = 100
	maxListLimit     = 500
)

type taskRecordRepositoryImpl struct {
	db *gorm.DB
}

func NewTaskRecordRepository(db *gorm.DB) TaskRecordRepository {
	return &taskRecordRepositoryImpl{db: db}
}

func (t *taskRecordRepositoryImpl) Record(ctx context.Context, r *model.TaskRecord) error {
	return t.db.WithContext(ctx).Create(r).Error
}

func (t *taskRecordRepositoryImpl) ListByTaskId(ctx context.Context, taskId string, limit, offset int) ([]*model.TaskRecord, error) {
	if limit <= 0 || limit > maxListLimit {
		limit = defaultListLimit
	}
	if offset < 0 {
		offset = 0
	}
	var out []*model.TaskRecord
	err := t.db.WithContext(ctx).
		Where("task_id = ?", taskId).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&out).Error
	return out, err
}
