package scheduler

import (
	"fmt"
	"os"
	"time"
)

// SchedulerConfig configures the scheduler and its timing wheel.
type SchedulerConfig struct {
	ScanInterval      time.Duration `json:"scan_interval" yaml:"ScanInterval"`
	LookaheadWindow   time.Duration `json:"lookahead_window" yaml:"LookaheadWindow"`
	BatchSize         int           `json:"batch_size" yaml:"BatchSize"`
	LeaseDuration     time.Duration `json:"lease_duration" yaml:"LeaseDuration"`
	WorkerConcurrency int           `json:"worker_concurrency" yaml:"WorkerConcurrency"`
	InstanceID        string        `json:"instance_id" yaml:"InstanceID"`

	// TimingWheel tuning (optional; defaults applied in ResolveSchedulerConfig).
	TickInterval  time.Duration `json:"tick_interval" yaml:"TickInterval"`
	SlotsPerLevel int           `json:"slots_per_level" yaml:"SlotsPerLevel"`
	MaxLevels     int           `json:"max_levels" yaml:"MaxLevels"`
}

func defaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		ScanInterval:      1 * time.Second,
		LookaheadWindow:   5 * time.Second,
		BatchSize:         100,
		LeaseDuration:     30 * time.Second,
		WorkerConcurrency: 8,
		TickInterval:      100 * time.Millisecond,
		SlotsPerLevel:     256,
		MaxLevels:         4,
	}
}

// ResolveSchedulerConfig fills zero-valued fields with defaults and assigns an
// InstanceID derived from hostname + pid when none is provided. The InstanceID is
// used for logging only (it is not written to the DB this round).
func ResolveSchedulerConfig(in SchedulerConfig) SchedulerConfig {
	d := defaultSchedulerConfig()
	if in.ScanInterval <= 0 {
		in.ScanInterval = d.ScanInterval
	}
	if in.LookaheadWindow <= 0 {
		in.LookaheadWindow = d.LookaheadWindow
	}
	if in.BatchSize <= 0 {
		in.BatchSize = d.BatchSize
	}
	if in.LeaseDuration <= 0 {
		in.LeaseDuration = d.LeaseDuration
	}
	if in.WorkerConcurrency <= 0 {
		in.WorkerConcurrency = d.WorkerConcurrency
	}
	if in.TickInterval <= 0 {
		in.TickInterval = d.TickInterval
	}
	if in.SlotsPerLevel < 2 {
		in.SlotsPerLevel = d.SlotsPerLevel
	}
	if in.MaxLevels < 1 {
		in.MaxLevels = d.MaxLevels
	}
	if in.InstanceID == "" {
		host, _ := os.Hostname()
		if host == "" {
			host = "unknown"
		}
		in.InstanceID = fmt.Sprintf("%s-%d", host, os.Getpid())
	}
	return in
}
