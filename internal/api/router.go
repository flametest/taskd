package api

import (
	"github.com/flametest/taskd/internal/api/handler"
	"github.com/flametest/taskd/internal/container"
	"github.com/flametest/taskd/internal/infra/repository"
	"github.com/flametest/taskd/internal/service"
	"github.com/flametest/vita/vserver"
)

type App struct {
	repository.Repository
}

func NewApp() *App {
	return &App{}
}

func (a *App) Router(server vserver.Server) vserver.Server {
	srv := server.(*vserver.EchoServer)
	e := srv.GetEchoServer()
	c := container.NewContainer(a.Repository)
	taskService := service.NewTaskService(c)
	taskHandler := handler.NewTaskHandler(taskService)
	e.Add("POST", "/v1/tasks", taskHandler.CreateTasks)
	return srv
}
