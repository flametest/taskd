package service

import (
	"github.com/flametest/taskd/internal/container"
	"github.com/flametest/taskd/internal/domain"
)

type TaskService interface {
	GetTaskById(id uint) (*domain.Task, error)
}

type taskServiceImpl struct {
	container container.Container
}

func NewTaskService(container container.Container) TaskService {
	return &taskServiceImpl{container: container}
}

func (t *taskServiceImpl) GetTaskById(id uint) (*domain.Task, error) {
	//TODO implement me
	panic("implement me")
}
