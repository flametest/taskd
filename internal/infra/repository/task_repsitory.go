package repository

import (
	"context"

	"github.com/flametest/taskd/internal/infra/model"
	"gorm.io/gorm"
)

type TaskRepository interface {
	Create(ctx context.Context, task *model.Task) error
	GetTaskById(ctx context.Context, taskId uint64) (*model.Task, error)
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
