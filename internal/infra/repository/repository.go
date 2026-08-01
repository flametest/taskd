package repository

import (
	"github.com/flametest/vita/vgorm"

	"gorm.io/gorm"
)

type Repository interface {
	GetTaskRepo(tx ...vgorm.Tx) TaskRepository
	GetTaskRecordRepo(tx ...vgorm.Tx) TaskRecordRepository
}

type repositoryImpl struct {
	taskRepo       TaskRepository
	taskRecordRepo TaskRecordRepository
}

func (r repositoryImpl) GetTaskRepo(tx ...vgorm.Tx) TaskRepository {
	if len(tx) == 0 || tx[0] == nil {
		return r.taskRepo
	}
	t := tx[0]
	return NewTaskRepository(t.DB())
}

func (r repositoryImpl) GetTaskRecordRepo(tx ...vgorm.Tx) TaskRecordRepository {
	if len(tx) == 0 || tx[0] == nil {
		return r.taskRecordRepo
	}
	t := tx[0]
	return NewTaskRecordRepository(t.DB())
}

func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{
		taskRepo:       NewTaskRepository(db),
		taskRecordRepo: NewTaskRecordRepository(db),
	}
}
