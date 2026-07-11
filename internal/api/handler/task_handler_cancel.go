package handler

import (
	"net/http"

	"github.com/flametest/vita/verrors"
	"github.com/labstack/echo/v4"
)

// CancelTask cancels a scheduled task via POST /v1/tasks/:id/cancel. Only tasks
// still in scheduled status can be canceled; claimed (executing) or finished
// tasks return ConflictError.
func (t *TaskHandler) CancelTask(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return verrors.BadRequestError("invalid task id")
	}
	if err := t.taskService.CancelTask(c.Request().Context(), id); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
