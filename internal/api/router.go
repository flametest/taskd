package api

import (
	"github.com/flametest/taskd/internal/api/handler"
	"github.com/flametest/taskd/internal/container"
	"github.com/flametest/taskd/internal/service"
	"github.com/flametest/vita/vserver"
)

type App struct {
	container.Container
}

func NewApp(c container.Container) *App {
	return &App{c}
}

func (a *App) Router(server vserver.Server) vserver.Server {
	srv := server.(*vserver.EchoServer)
	e := srv.GetEchoServer()
	taskService := service.NewTaskService(a.Container)
	taskHandler := handler.NewTaskHandler(taskService)
	e.Add("POST", "/v1/tasks", taskHandler.CreateTasks)
	e.Add("GET", "/v1/tasks/:id", taskHandler.GetTaskById)
	e.Add("GET", "/v1/tasks/:id/records", taskHandler.ListTaskRecords)
	return srv
}
