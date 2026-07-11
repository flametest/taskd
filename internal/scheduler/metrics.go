package scheduler

import (
	"github.com/flametest/vita/vmetrics"
	"github.com/prometheus/client_golang/prometheus"
)

// Scheduler business metrics, registered against vmetrics.Registry (exposed at
// /metrics via the WithMetrics server option). Declared once at package load.
var (
	tasksClaimed = vmetrics.NewCounterVec(prometheus.CounterOpts{
		Name: "taskd_tasks_claimed_total",
		Help: "Total number of tasks claimed from the DB by the scan loop.",
	}, nil)

	executionsTotal = vmetrics.NewCounterVec(prometheus.CounterOpts{
		Name: "taskd_executions_total",
		Help: "Total task executions, partitioned by result (success/failure).",
	}, []string{"result"})

	executionDuration = vmetrics.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "taskd_execution_duration_seconds",
		Help:    "Time spent in executor.Execute per attempt, in seconds.",
		Buckets: prometheus.DefBuckets,
	}, nil)

	tasksReclaimed = vmetrics.NewCounterVec(prometheus.CounterOpts{
		Name: "taskd_reclaimed_total",
		Help: "Total claimed tasks reset to scheduled by the lease reaper.",
	}, nil)

	workerQueueDepth = vmetrics.NewGauge(prometheus.GaugeOpts{
		Name: "taskd_worker_queue_depth",
		Help: "Current number of tasks queued in the worker-pool channel.",
	})
)
