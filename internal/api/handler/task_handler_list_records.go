package handler

import (
	"net/http"
	"strconv"

	"github.com/flametest/taskd/pkg/dto"
	"github.com/flametest/vita/verrors"
	"github.com/labstack/echo/v4"
)

// ListTaskRecords returns the execution-audit history of one task, newest-first,
// via GET /v1/tasks/:id/records?limit=&offset=.
func (t *TaskHandler) ListTaskRecords(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return verrors.BadRequestError("invalid task id")
	}
	limit, offset := parseLimitOffset(c)
	records, err := t.taskService.ListTaskRecords(c.Request().Context(), id, limit, offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, dto.ListTaskRecordsResp{
		Records: records,
		Limit:   limit,
		Offset:  offset,
	})
}

// parseLimitOffset reads the optional limit/offset query params with sane defaults
// and mild clamping. The repository re-clamps defensively.
func parseLimitOffset(c echo.Context) (limit, offset int) {
	limit = 100
	offset = 0
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}
