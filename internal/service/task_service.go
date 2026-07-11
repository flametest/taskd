package service

import (
	"context"
	"time"

	"github.com/flametest/taskd/internal/container"
	"github.com/flametest/taskd/internal/domain"
	"github.com/flametest/taskd/internal/infra/model"
	"github.com/flametest/taskd/pkg/dto"
)

type TaskService interface {
	GetTaskById(ctx context.Context, id string) (*domain.Task, error)
	CreateTask(ctx context.Context, req *dto.CreatTaskReq) (*domain.Task, error)
	ListTaskRecords(ctx context.Context, taskId string, limit, offset int) ([]*model.TaskRecord, error)
	// ReactivateTask reactivates a dead task, re-scheduling it at execTime (or
	// now when nil). Returns ConflictError if the task is not in dead status.
	ReactivateTask(ctx context.Context, taskId string, execTime *time.Time) error
	// CancelTask cancels a scheduled task. Returns ConflictError if the task is
	// not in scheduled status (claimed/executing/finished tasks cannot be canceled).
	CancelTask(ctx context.Context, taskId string) error
}

type taskServiceImpl struct {
	container container.Container
}

func NewTaskService(container container.Container) TaskService {
	return &taskServiceImpl{container: container}
}

func (t *taskServiceImpl) GetTaskById(ctx context.Context, id string) (*domain.Task, error) {
	taskDO, err := t.container.GetRepository().GetTaskRepo().GetTaskById(ctx, id)
	if err != nil {
		return nil, err
	}
	return domain.NewFromDO(taskDO), nil
}

func (t *taskServiceImpl) CreateTask(ctx context.Context, req *dto.CreatTaskReq) (*domain.Task, error) {
	task := domain.NewTask(req.Body.Name, req.Body.RefId, req.Body.Protocol, req.Body.Address, req.Body.Params, req.Body.ExecTime,
		req.Body.MaxRetries)
	taskDO := task.ToDO()
	err := t.container.GetRepository().GetTaskRepo().Create(ctx, taskDO)
	if err != nil {
		return nil, err
	}
	task.SetId(taskDO.Id)
	return task, nil
}

func (t *taskServiceImpl) ListTaskRecords(ctx context.Context, taskId string, limit, offset int) ([]*model.TaskRecord, error) {
	return t.container.GetRepository().GetTaskRecordRepo().ListByTaskId(ctx, taskId, limit, offset)
}

func (t *taskServiceImpl) ReactivateTask(ctx context.Context, taskId string, execTime *time.Time) error {
	next := time.Now()
	if execTime != nil {
		next = *execTime
	}
	return t.container.GetRepository().GetTaskRepo().Reactivate(ctx, taskId, next)
}

func (t *taskServiceImpl) CancelTask(ctx context.Context, taskId string) error {
	return t.container.GetRepository().GetTaskRepo().Cancel(ctx, taskId)
}
