package handler

import (
	"net/http"

	"github.com/flametest/taskd/pkg/dto"
	"github.com/flametest/vita/verrors"
	"github.com/labstack/echo/v4"
)

// ReactivateTask reactivates a dead task via POST /v1/tasks/:id/reactivate. An
// optional {"body":{"exec_time":<unix>}} re-schedules it at a future time; an
// empty body runs it immediately.
func (t *TaskHandler) ReactivateTask(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return verrors.BadRequestError("invalid task id")
	}
	req := &dto.ReactivateTaskReq{}
	// Body is optional (all fields optional); ignore bind errors for an empty body.
	_ = (&echo.DefaultBinder{}).BindBody(c, &req.Body)
	if err := t.taskService.ReactivateTask(c.Request().Context(), id, req.Body.ExecTimeAsTime()); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
