package handler

import (
	"net/http"

	"github.com/flametest/taskd/pkg/dto"
	"github.com/flametest/vita/verrors"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

func (t *TaskHandler) CreateTasks(c echo.Context) error {
	req := &dto.CreatTaskReq{}
	binder := echo.DefaultBinder{}
	err := binder.BindBody(c, &req.Body)
	if err != nil {
		return verrors.BadRequestError(err.Error())
	}
	validate := validator.New()
	err = validate.Struct(req)
	if err != nil {
		return verrors.BadRequestError(err.Error())
	}
	task, err := t.taskService.CreateTask(c.Request().Context(), req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, task)
}
