package dto

import (
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
)

// Task is the API representation of a task. Unlike domain.Task it exposes Status
// (the domain keeps it private) and represents LastError as a string, so the
// JSON response includes the task's current state.
type Task struct {
	Id          string                 `json:"id"`
	Name        string                 `json:"name"`
	RefId       string                 `json:"ref_id"`
	Protocol    enum.Protocol          `json:"protocol"`
	Address     string                 `json:"address"`
	Params      map[string]interface{} `json:"params"`
	ExecTime    *time.Time             `json:"exec_time"`
	Status      enum.Status            `json:"status"`
	Attempts    int                    `json:"attempts"`
	MaxRetries  int                    `json:"max_retries"`
	LastError   string                 `json:"last_error"`
	LockedUntil *time.Time             `json:"locked_until"`
	Cron        string                 `json:"cron"`
	CreatedAt   time.Time              `json:"created_at"`
}

// ListTasksResp is the response of GET /v1/tasks.
type ListTasksResp struct {
	Tasks  []*Task `json:"tasks"`
	Total  int64   `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}
