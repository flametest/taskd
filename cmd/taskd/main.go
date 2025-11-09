package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"
	"time"

	"github.com/flametest/taskd/internal/api"
	"github.com/flametest/taskd/internal/config"
	"github.com/flametest/vita/vlog"
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
	log.InitLogger(cfg.AppConfig.Name, cfg.LogLevel)
	log.Info().Msg("starting taskd")
	srv, err := vserver.NewEchoServer(ctx, &cfg.AppConfig)
	if err != nil {
		panic(err)
	}
	app := api.NewApp()
	srv.Register(app.Router)
	go func() {
		_ = srv.Start(ctx)
	}()

	<-ctx.Done()

	log.Info().Msg("shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}
	log.Info().Msg("Server exiting")
}
