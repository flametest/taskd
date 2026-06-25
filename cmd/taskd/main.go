package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"
	"time"

	"github.com/flametest/taskd/internal/api"
	"github.com/flametest/taskd/internal/config"
	"github.com/flametest/taskd/internal/container"
	"github.com/flametest/taskd/internal/scheduler"
	"github.com/flametest/taskd/pkg/timingwheel"
	"github.com/flametest/vita/verrors"
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
	verrors.Initialize(cfg.AppConfig.Name)
	log.InitLogger(log.ZerologType, cfg.AppConfig.Name, cfg.LogLevel)
	log.Info().Msg("starting taskd")
	srv, err := vserver.NewEchoServer(ctx, &cfg.AppConfig)
	if err != nil {
		panic(err)
	}
	c, err := container.NewContainer(cfg)
	if err != nil {
		panic(err)
	}
	app := api.NewApp(c)
	srv.Register(app.Router)
	go func() {
		_ = srv.Start(ctx)
	}()

	// Start the task scheduler (optional; disabled when no Scheduler config).
	var sched *scheduler.Scheduler
	if cfg.Scheduler != nil {
		resolved := scheduler.ResolveSchedulerConfig(*cfg.Scheduler)
		wheel := timingwheel.New(
			timingwheel.WithTickInterval(resolved.TickInterval),
			timingwheel.WithSlotsPerLevel(resolved.SlotsPerLevel),
			timingwheel.WithMaxLevels(resolved.MaxLevels),
		)
		sched = scheduler.NewScheduler(resolved, c.GetRepository().GetTaskRepo(), wheel, scheduler.NewNoopExecutor())
		sched.Start(ctx)
		log.Info().Any("instance_id", resolved.InstanceID).Msg("scheduler started")
	} else {
		log.Info().Msg("scheduler disabled (no Scheduler config)")
	}

	<-ctx.Done()

	log.Info().Msg("shutting down gracefully...")

	// Stop the scheduler first: stop claiming, stop the wheel, drain workers.
	if sched != nil {
		sched.Stop()
		log.Info().Msg("scheduler stopped")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Any("error", err).Msg("Server forced to shutdown")
	}
	log.Info().Msg("Server exiting")
}
