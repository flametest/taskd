package service

import (
	"context"

	"github.com/flametest/taskd/internal/container"
	"github.com/flametest/taskd/internal/domain"
	"github.com/flametest/taskd/pkg/dto"
)

type TaskService interface {
	GetTaskById(ctx context.Context, id uint) (*domain.Task, error)
	CreateTask(ctx context.Context, req *dto.CreatTaskReq) (*domain.Task, error)
}

type taskServiceImpl struct {
	container container.Container
}

func NewTaskService(container container.Container) TaskService {
	return &taskServiceImpl{container: container}
}

func (t *taskServiceImpl) GetTaskById(ctx context.Context, id uint) (*domain.Task, error) {
	//TODO implement me
	panic("implement me")
}

func (t *taskServiceImpl) CreateTask(ctx context.Context, req *dto.CreatTaskReq) (*domain.Task, error) {
	//TODO implement me
	panic("implement me")
}
