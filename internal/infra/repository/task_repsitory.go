package repository

import (
	"context"

	"github.com/flametest/taskd/internal/infra/model"
	"gorm.io/gorm"
)

type TaskRepository interface {
	Create(ctx context.Context, task *model.Task) error
}

type taskRepositoryImpl struct {
	db *gorm.DB
}

func (t taskRepositoryImpl) Create(ctx context.Context, task *model.Task) error {
	//TODO implement me
	panic("implement me")
}
