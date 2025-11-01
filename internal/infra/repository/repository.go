package repository

import (
	"github.com/flametest/vita/vgorm"

	"gorm.io/gorm"
)

type Repository interface {
	GetTaskRepo(tx ...vgorm.Tx) TaskRepository
}

type repositoryImpl struct {
	db       *gorm.DB
	taskRepo TaskRepository
}

func (r repositoryImpl) GetTaskRepo(tx ...vgorm.Tx) TaskRepository {
	if len(tx) == 0 || tx[0] == nil {
		return r.taskRepo
	}
	t, ok := tx[0].(vgorm.Tx)
	if !ok {
		return r.taskRepo
	}
	return NewTaskRepository(t.DB())
}

func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{
		taskRepo: NewTaskRepository(db),
	}
}
