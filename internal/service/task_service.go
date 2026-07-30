package service

import (
	"context"
	"fmt"
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/internal/container"
	cronpkg "github.com/flametest/taskd/internal/cron"
	"github.com/flametest/taskd/internal/domain"
	"github.com/flametest/taskd/internal/infra/model"
	"github.com/flametest/taskd/pkg/dto"
	"github.com/flametest/vita/verrors"
)

type TaskService interface {
	GetTaskById(ctx context.Context, id string) (*domain.Task, error)
	CreateTask(ctx context.Context, req *dto.CreatTaskReq) (*domain.Task, error)
	ListTasks(ctx context.Context, status *enum.Status, limit, offset int) ([]*domain.Task, error)
	ListTaskRecords(ctx context.Context, taskId string, limit, offset int) ([]*model.TaskRecord, error)
	// ReactivateTask reactivates a dead task, re-scheduling it at execTime (or
	// now when nil). Returns ConflictError if the task is not in dead status.
	ReactivateTask(ctx context.Context, taskId string, execTime *time.Time) error
	// CancelTask cancels a scheduled or claimed (running) task. Canceling a
	// running task stops further scheduling but does not interrupt the in-flight
	// execution. Returns ConflictError for terminal states (succeeded/dead/canceled).
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
	if req.Body.Cron != "" {
		if err := cronpkg.Validate(req.Body.Cron); err != nil {
			return nil, verrors.BadRequestError(fmt.Sprintf("invalid cron: %v", err))
		}
		// Cron governs subsequent occurrences; the first run time must be supplied
		// explicitly (the service does not know the scheduler's timezone to compute it).
		if req.Body.ExecTime == 0 {
			return nil, verrors.BadRequestError("exec_time is required for cron tasks")
		}
	}
	task := domain.NewTask(req.Body.Name, req.Body.RefId, req.Body.Protocol, req.Body.Address, req.Body.Params, req.Body.ExecTime,
		req.Body.MaxRetries)
	if req.Body.Cron != "" {
		task.SetCron(req.Body.Cron)
	}
	taskDO := task.ToDO()
	err := t.container.GetRepository().GetTaskRepo().Create(ctx, taskDO)
	if err != nil {
		return nil, err
	}
	task.SetId(taskDO.Id)
	return task, nil
}

func (t *taskServiceImpl) ListTasks(ctx context.Context, status *enum.Status, limit, offset int) ([]*domain.Task, error) {
	tasks, err := t.container.GetRepository().GetTaskRepo().ListTasks(ctx, status, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Task, 0, len(tasks))
	for _, taskDO := range tasks {
		out = append(out, domain.NewFromDO(taskDO))
	}
	return out, nil
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
