package main

import (
	"context"
	"flag"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/flametest/taskd/internal/config"
	"github.com/flametest/vita/vserver"
)

var cfgFile = flag.String("config", "deploy/server-config.yaml", "config file")

func main() {
	var err error
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGHUP,
	)
	defer stop()

	cfg, err := config.ParseConfig(*cfgFile)
	if err != nil {
		panic(err)
	}
	srv, err := vserver.NewEchoServer(ctx, &cfg.AppConfig)
	if err != nil {
		panic(err)
	}

	go func() {
		_ = srv.Start(ctx)
	}()

	<-ctx.Done()

	fmt.Println("shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Println("Server forced to shutdown: ", err)
	}
	fmt.Println("Server exiting")
}
