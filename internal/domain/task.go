package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/internal/infra/model"
	"github.com/flametest/vita/verrors"
	"github.com/flametest/vita/vgorm"
	log "github.com/flametest/vita/vlog"
	"github.com/samber/lo"
)

type Task struct {
	Id          string
	Name        string
	RefId       string
	Protocol    enum.Protocol
	Address     string
	Params      map[string]interface{}
	ExecTime    *time.Time
	status      enum.Status
	Attempts    int
	MaxRetries  int
	LastError   error
	LockedUntil *time.Time
	Cron        string
	CreatedAt   time.Time
}

func NewTask(name, refId string, protocol enum.Protocol, address string, params map[string]interface{}, execTime int64,
	maxRetries int) *Task {
	if maxRetries == 0 {
		maxRetries = 5
	}
	return &Task{
		Name:       name,
		RefId:      refId,
		Protocol:   protocol,
		Address:    address,
		Params:     params,
		ExecTime:   lo.ToPtr(time.Unix(execTime, 0)),
		status:     enum.TaskStatusScheduled,
		Attempts:   0,
		MaxRetries: maxRetries,
		LastError:  nil,
	}
}

func (t *Task) SetId(id string) *Task {
	t.Id = id
	return t
}

// SetCron sets the cron expression for a recurring task. Empty means one-shot.
func (t *Task) SetCron(c string) *Task {
	t.Cron = c
	return t
}

// Claim transitions the task from 'scheduled' to 'claimed' and records the lease
// expiry. It is an error to claim a task that is not 'scheduled'.
func (t *Task) Claim(until time.Time) error {
	if t.status != enum.TaskStatusScheduled {
		return verrors.ConflictError(fmt.Sprintf("task %s not scheduled (current: %s)", t.Id, t.status))
	}
	t.status = enum.TaskStatusClaimed
	t.LockedUntil = &until
	return nil
}

// MarkSucceeded transitions the task from 'claimed' to 'succeeded'.
func (t *Task) MarkSucceeded() error {
	if t.status != enum.TaskStatusClaimed {
		return verrors.ConflictError(fmt.Sprintf("task %s not claimed (current: %s)", t.Id, t.status))
	}
	t.status = enum.TaskStatusSucceeded
	return nil
}

// Status exposes the current status (read-only).
func (t *Task) Status() enum.Status { return t.status }

func NewFromDO(do *model.Task) *Task {
	var params map[string]interface{}
	if do.Params != nil {
		err := json.Unmarshal(do.Params, &params)
		if err != nil {
			log.Panic().Msgf("error unmarshalling params: %v", err)
		}
	}

	var lastError error
	if do.LastError != "" {
		lastError = fmt.Errorf("%s", do.LastError)
	}

	return &Task{
		Id:          do.Id,
		Name:        do.Name,
		RefId:       do.RefId,
		Protocol:    do.Protocol,
		Address:     do.Address,
		Params:      params,
		ExecTime:    lo.ToPtr(do.ExecTime),
		status:      do.Status,
		Attempts:    do.Attempts,
		MaxRetries:  do.MaxRetries,
		LastError:   lastError,
		LockedUntil: do.LockedUntil,
		Cron:        do.Cron,
		CreatedAt:   do.CreatedAt,
	}
}

func (t *Task) ToDO() *model.Task {
	var lastErrorStr string
	if t.LastError != nil {
		lastErrorStr = t.LastError.Error()
	}
	paramsJson, err := json.Marshal(t.Params)
	if err != nil {
		return nil
	}
	return &model.Task{
		BasePostgres: vgorm.BasePostgres{
			Id: t.Id,
		},
		Name:        t.Name,
		RefId:       t.RefId,
		Protocol:    t.Protocol,
		Address:     t.Address,
		Params:      paramsJson,
		ExecTime:    *t.ExecTime,
		Status:      t.status,
		Attempts:    t.Attempts,
		MaxRetries:  t.MaxRetries,
		LastError:   lastErrorStr,
		LockedUntil: t.LockedUntil,
		Cron:        t.Cron,
	}
}
