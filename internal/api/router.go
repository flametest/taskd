package api

import (
	"github.com/flametest/taskd/internal/api/handler"
	"github.com/flametest/vita/vserver"
)

func Router(server vserver.Server) vserver.Server {
	srv := server.(*vserver.EchoServer)
	e := srv.GetEchoServer()

	taskHandler := handler.NewTaskHandler()
	e.Add("POST", "/v1/tasks", taskHandler.CreateTasks)
	return srv
}
