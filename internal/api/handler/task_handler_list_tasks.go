package handler

import (
	"net/http"
	"time"

	"github.com/flametest/taskd/internal/api/handler/converter"
	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/internal/infra/repository"
	"github.com/flametest/taskd/pkg/dto"
	"github.com/labstack/echo/v4"
)

// ListTasks returns tasks via GET /v1/tasks?status=&search=&created_from=&created_to=&limit=&offset=,
// ordered by exec_time ascending. status/search/created_from/created_to are all optional.
func (t *TaskHandler) ListTasks(c echo.Context) error {
	limit, offset := parseLimitOffset(c)
	filter := repository.TaskFilter{}
	if s := c.QueryParam("status"); s != "" {
		st := enum.Status(s)
		filter.Status = &st
	}
	if s := c.QueryParam("search"); s != "" {
		filter.Search = s
	}
	if s := c.QueryParam("created_from"); s != "" {
		if parsed, err := time.Parse(time.RFC3339, s); err == nil {
			filter.CreatedFrom = &parsed
		}
	}
	if s := c.QueryParam("created_to"); s != "" {
		if parsed, err := time.Parse(time.RFC3339, s); err == nil {
			filter.CreatedTo = &parsed
		}
	}
	tasks, total, err := t.taskService.ListTasks(c.Request().Context(), filter, limit, offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, dto.ListTasksResp{Tasks: converter.ToTasks(tasks), Total: total, Limit: limit, Offset: offset})
}
