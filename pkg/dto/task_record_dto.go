package dto

import (
	"encoding/json"
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
)

// TaskRecord is the API representation of a task_record row. model.TaskRecord has
// no JSON tags (it would serialize as PascalCase), so this DTO gives stable
// snake_case wire field names that match the frontend types.
type TaskRecord struct {
	Id           string          `json:"id"`
	TaskId       string          `json:"task_id"`
	Attempt      int             `json:"attempt"`
	Result       enum.Result     `json:"result"`
	Protocol     enum.Protocol   `json:"protocol"`
	InstanceId   string          `json:"instance_id"`
	ErrorMessage string          `json:"error_message"`
	StartedAt    time.Time       `json:"started_at"`
	FinishedAt   time.Time       `json:"finished_at"`
	DurationMs   int64           `json:"duration_ms"`
	Response     json.RawMessage `json:"response,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// ListTaskRecordsResp is the response of GET /v1/tasks/:id/records: one task's
// execution-audit rows, newest-first, with the applied pagination echoed back. No
// total count — the table is append-only and a COUNT(*) per request is wasteful;
// clients paginate by offset until a page returns fewer than limit rows.
type ListTaskRecordsResp struct {
	Records []*TaskRecord `json:"records"`
	Limit   int           `json:"limit"`
	Offset  int           `json:"offset"`
}
