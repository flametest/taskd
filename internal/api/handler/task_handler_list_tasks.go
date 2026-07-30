package handler

import (
	"net/http"

	"github.com/flametest/taskd/internal/api/handler/converter"
	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/pkg/dto"
	"github.com/labstack/echo/v4"
)

// ListTasks returns tasks via GET /v1/tasks?status=&limit=&offset=, ordered by
// exec_time ascending. status is optional (omit for all statuses).
func (t *TaskHandler) ListTasks(c echo.Context) error {
	limit, offset := parseLimitOffset(c)
	var status *enum.Status
	if s := c.QueryParam("status"); s != "" {
		st := enum.Status(s)
		status = &st
	}
	tasks, err := t.taskService.ListTasks(c.Request().Context(), status, limit, offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, dto.ListTasksResp{Tasks: converter.ToTasks(tasks), Limit: limit, Offset: offset})
}
