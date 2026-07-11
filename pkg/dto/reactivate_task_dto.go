package dto

import "time"

// ReactivateTaskReq is the body of POST /v1/tasks/:id/reactivate. All fields are
// optional; an empty body reactivates the task to run immediately.
type ReactivateTaskReq struct {
	Body ReactivateTaskReqBody `json:"body"`
}

type ReactivateTaskReqBody struct {
	// ExecTime is an optional Unix-second time to re-schedule the task at; when
	// omitted the task runs immediately (exec_time = now).
	ExecTime *int64 `json:"exec_time"`
}

// ExecTimeAsTime converts the optional Unix-second exec_time to a *time.Time,
// returning nil when unset.
func (b ReactivateTaskReqBody) ExecTimeAsTime() *time.Time {
	if b.ExecTime == nil {
		return nil
	}
	t := time.Unix(*b.ExecTime, 0)
	return &t
}
