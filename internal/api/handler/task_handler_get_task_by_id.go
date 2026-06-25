package handler

import (
	"net/http"

	"github.com/flametest/vita/verrors"
	"github.com/labstack/echo/v4"
)

func (t *TaskHandler) GetTaskById(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return verrors.BadRequestError("invalid task id")
	}

	task, err := t.taskService.GetTaskById(c.Request().Context(), id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, task)
}
