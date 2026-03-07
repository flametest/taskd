package handler

import (
	"net/http"
	"strconv"

	"github.com/flametest/vita/verrors"
	"github.com/labstack/echo/v4"
)

func (t *TaskHandler) GetTaskById(c echo.Context) error {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return verrors.BadRequestError("invalid task id")
	}

	task, err := t.taskService.GetTaskById(c.Request().Context(), id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, task)
}
