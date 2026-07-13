package handler

import (
	"net/http"

	"github.com/flametest/vita/verrors"
	"github.com/labstack/echo/v4"
)

// CancelTask cancels a scheduled or claimed (running) task via POST
// /v1/tasks/:id/cancel. Canceling a running task stops further scheduling but
// does not interrupt the in-flight execution (it finishes on its own). Tasks in
// terminal states (succeeded/dead/canceled) return ConflictError.
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
