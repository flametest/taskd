package dto

import "github.com/flametest/taskd/internal/infra/model"

// ListTaskRecordsResp is the response of GET /v1/tasks/:id/records: one task's
// execution-audit rows, newest-first, with the applied pagination echoed back. No
// total count — the table is append-only and a COUNT(*) per request is wasteful;
// clients paginate by offset until a page returns fewer than limit rows.
type ListTaskRecordsResp struct {
	Records []*model.TaskRecord `json:"records"`
	Limit   int                 `json:"limit"`
	Offset  int                 `json:"offset"`
}
