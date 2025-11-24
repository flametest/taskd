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
	task := domain.NewTask(req.Body.Name, req.Body.Protocol, req.Body.Address, req.Body.Params, req.Body.ExecTime,
		req.Body.MaxRetries)
	taskDO := task.ToDO()
	err := t.container.GetRepository().GetTaskRepo().Create(ctx, taskDO)
	if err != nil {
		return nil, err
	}
	task.SetId(taskDO.Id)
	return task, nil
}
