package domain

import (
	"encoding/json"
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/internal/infra/model"
	"github.com/flametest/vita/vtool"
	"github.com/google/uuid"
)

type Task struct {
	Id         uint64
	Name       string
	TaskId     string
	Protocol   enum.Protocol
	Address    string
	Params     map[string]interface{}
	ExecTime   *time.Time
	status     enum.Status
	Attempts   int
	MaxRetries int
	LastError  error
}

func NewTask(name string, protocol enum.Protocol, address string, params map[string]interface{}, execTime int64,
	maxRetries int) *Task {
	if maxRetries == 0 {
		maxRetries = 5
	}
	return &Task{
		Name:       name,
		TaskId:     uuid.New().String(),
		Protocol:   protocol,
		Address:    address,
		Params:     params,
		ExecTime:   vtool.Ptr(time.Unix(execTime, 0)),
		status:     enum.TaskStatusScheduled,
		Attempts:   0,
		MaxRetries: maxRetries,
		LastError:  nil,
	}
}

func (t *Task) SetId(id uint64) *Task {
	t.Id = id
	return t
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
		Base: model.Base{
			Id: t.Id,
		},
		Name:       t.Name,
		TaskId:     t.TaskId,
		Protocol:   t.Protocol,
		Address:    t.Address,
		Params:     paramsJson,
		ExecTime:   *t.ExecTime,
		Status:     t.status,
		Attempts:   t.Attempts,
		MaxRetries: t.MaxRetries,
		LastError:  lastErrorStr,
	}
}
