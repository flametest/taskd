package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/internal/infra/model"
	"github.com/flametest/vita/verrors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TaskRepository interface {
	Create(ctx context.Context, task *model.Task) error
	GetTaskById(ctx context.Context, taskId uint64) (*model.Task, error)
	// GetTasksByStatus returns up to limit tasks in the given status ordered by
	// exec_time ascending. Mostly a test/inspection helper.
	GetTasksByStatus(ctx context.Context, status enum.Status, limit int) ([]*model.Task, error)
	// Claim atomically claims up to batchSize scheduled tasks whose exec_time is
	// within now+lookahead, flipping them to 'claimed' and setting locked_until to
	// now+lease. It runs SELECT ... FOR UPDATE SKIP LOCKED -> UPDATE inside one
	// transaction so concurrent instances claim disjoint sets. Returns (nil, nil)
	// when nothing is due.
	// NOTE: the SKIP LOCKED guarantee is MySQL 8+ only; SQLite (used in unit
	// tests) does not support it, so multi-instance correctness is validated
	// manually against MySQL.
	Claim(ctx context.Context, now time.Time, lookahead time.Duration, batchSize int, lease time.Duration) ([]*model.Task, error)
	// MarkSucceeded flips a task from 'claimed' to 'succeeded'. It returns
	// ConflictError (rows affected 0) if the task is not in 'claimed' state,
	// guarding against double-fire and stale rows.
	MarkSucceeded(ctx context.Context, taskId string) error
}

type taskRepositoryImpl struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepositoryImpl{db: db}
}

func (t *taskRepositoryImpl) Create(ctx context.Context, task *model.Task) error {
	return t.db.WithContext(ctx).Create(task).Error
}

func (t *taskRepositoryImpl) GetTaskById(ctx context.Context, taskId uint64) (*model.Task, error) {
	var task model.Task
	err := t.db.WithContext(ctx).Where("id = ?", taskId).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (t *taskRepositoryImpl) GetTasksByStatus(ctx context.Context, status enum.Status, limit int) ([]*model.Task, error) {
	var out []*model.Task
	err := t.db.WithContext(ctx).
		Where("status = ?", status).
		Order("exec_time ASC").
		Limit(limit).
		Find(&out).Error
	return out, err
}

// Claim selects due scheduled rows with FOR UPDATE SKIP LOCKED, then updates them
// to 'claimed' with a lease, all inside one transaction. MySQL 8+ only.
func (t *taskRepositoryImpl) Claim(ctx context.Context, now time.Time, lookahead time.Duration,
	batchSize int, lease time.Duration) ([]*model.Task, error) {

	cutoff := now.Add(lookahead)
	leaseUntil := now.Add(lease)

	var claimed []*model.Task
	err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Step 1: lock candidate rows, skipping rows already locked by other instances.
		if err := tx.Clauses(clause.Locking{
			Strength: clause.LockingStrengthUpdate,
			Options:  clause.LockingOptionsSkipLocked,
		}).
			Where("status = ? AND exec_time <= ?", enum.TaskStatusScheduled, cutoff).
			Order("exec_time ASC").
			Limit(batchSize).
			Find(&claimed).Error; err != nil {
			return err
		}
		if len(claimed) == 0 {
			return nil
		}

		// Step 2: flip status + record lease on the locked rows. Map form so GORM
		// writes every column (no zero-value skipping).
		ids := make([]string, 0, len(claimed))
		for _, c := range claimed {
			ids = append(ids, c.Id)
		}
		if err := tx.Model(&model.Task{}).
			Where("id IN ?", ids).
			Updates(map[string]interface{}{
				"status":       enum.TaskStatusClaimed,
				"locked_until": leaseUntil,
			}).Error; err != nil {
			return err
		}

		// Reflect the mutation in the returned models so the caller need not re-fetch.
		for _, c := range claimed {
			c.Status = enum.TaskStatusClaimed
			c.LockedUntil = &leaseUntil
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

// MarkSucceeded is guarded by status='claimed'; a no-op row (already
// succeeded/failed/stale) yields ConflictError.
func (t *taskRepositoryImpl) MarkSucceeded(ctx context.Context, taskId string) error {
	res := t.db.WithContext(ctx).
		Model(&model.Task{}).
		Where("id = ? AND status = ?", taskId, enum.TaskStatusClaimed).
		Update("status", enum.TaskStatusSucceeded)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return verrors.ConflictError(fmt.Sprintf("task %s not in claimed state", taskId))
	}
	return nil
}
