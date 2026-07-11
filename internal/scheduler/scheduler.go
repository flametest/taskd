package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/internal/domain"
	"github.com/flametest/taskd/internal/infra/model"
	"github.com/flametest/taskd/pkg/timingwheel"
	log "github.com/flametest/vita/vlog"
	"gorm.io/datatypes"
)

// taskClaimer is the narrow slice of repository.TaskRepository the scheduler
// needs, declared locally so tests can inject a fake without mocking the whole
// repository.
type taskClaimer interface {
	Claim(ctx context.Context, now time.Time, lookahead time.Duration, batchSize int, lease time.Duration) ([]*model.Task, error)
	MarkSucceeded(ctx context.Context, taskId string) error
	MarkFailure(ctx context.Context, taskId string, lastError string, nextExecTime time.Time) (int64, error)
	MarkDead(ctx context.Context, taskId string, lastError string) error
	ReclaimOrphans(ctx context.Context, now time.Time) (int64, error)
}

// taskRecorder is the narrow slice of repository.TaskRecordRepository the
// scheduler needs to append execution-audit rows. Kept separate from taskClaimer
// because recording is best-effort and orthogonal to the task state machine. A
// nil recorder (the default) disables auditing.
type taskRecorder interface {
	Record(ctx context.Context, r *model.TaskRecord) error
}

// Scheduler claims due tasks, hands them to a TimingWheel for precise firing, and
// executes fired tasks on a fixed-size worker pool. Round 1 closes the loop
// scheduled -> claimed -> succeeded; the failure/retry path is a later round.
type Scheduler struct {
	cfg      SchedulerConfig
	repo     taskClaimer
	recorder taskRecorder
	wheel    timingwheel.TimingWheel
	exec     Executor

	taskCh   chan *model.Task
	wg       sync.WaitGroup
	scanWG   sync.WaitGroup
	reaperWG sync.WaitGroup

	started atomic.Bool
	stopCh  chan struct{}
}

// NewScheduler constructs a scheduler. repo is typically a repository.TaskRepository
// (it satisfies the taskClaimer interface).
func NewScheduler(cfg SchedulerConfig, repo taskClaimer, wheel timingwheel.TimingWheel, exec Executor) *Scheduler {
	return &Scheduler{
		cfg:    cfg,
		repo:   repo,
		wheel:  wheel,
		exec:   exec,
		taskCh: make(chan *model.Task, cfg.BatchSize),
	}
}

// WithRecorder installs an execution-audit recorder. Optional; a nil recorder
// (the default) disables auditing. Returns the scheduler for chaining.
func (s *Scheduler) WithRecorder(r taskRecorder) *Scheduler {
	s.recorder = r
	return s
}

// Start launches the scan loop and worker pool. Idempotent.
func (s *Scheduler) Start(ctx context.Context) {
	if !s.started.CompareAndSwap(false, true) {
		return
	}
	s.stopCh = make(chan struct{})
	s.wheel.Start()
	for i := 0; i < s.cfg.WorkerConcurrency; i++ {
		s.wg.Add(1)
		go s.workerLoop()
	}
	s.scanWG.Add(1)
	go s.scanLoop(ctx)
	s.reaperWG.Add(1)
	go s.reaperLoop(ctx)
}

// reaperLoop periodically reclaims tasks whose lease has expired.
func (s *Scheduler) reaperLoop(ctx context.Context) {
	defer s.reaperWG.Done()
	ticker := time.NewTicker(s.cfg.ReaperInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.reapOnce(ctx)
		}
	}
}

func (s *Scheduler) reapOnce(ctx context.Context) {
	n, err := s.repo.ReclaimOrphans(ctx, time.Now())
	if err != nil {
		log.Error().Any("error", err).Msg("scheduler: reap failed")
		return
	}
	if n > 0 {
		tasksReclaimed.WithLabelValues().Add(float64(n))
		log.Info().Any("reclaimed", n).Msg("scheduler: reclaimed orphaned tasks")
	}
}

// Stop shuts the scheduler down: stops claiming, stops the wheel (no new fires),
// then drains the worker pool so every task that reached the channel executes.
// Idempotent and safe to call before Start.
func (s *Scheduler) Stop() {
	if !s.started.CompareAndSwap(true, false) {
		return
	}
	close(s.stopCh)
	s.scanWG.Wait()
	s.reaperWG.Wait()
	s.wheel.Stop()
	s.wg.Wait()
}

func (s *Scheduler) scanLoop(ctx context.Context) {
	defer s.scanWG.Done()
	ticker := time.NewTicker(s.cfg.ScanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.scanOnce(ctx)
		}
	}
}

func (s *Scheduler) scanOnce(ctx context.Context) {
	now := time.Now()
	claimed, err := s.repo.Claim(ctx, now, s.cfg.LookaheadWindow, s.cfg.BatchSize, s.cfg.LeaseDuration)
	if err != nil {
		log.Error().Any("error", err).Msg("scheduler: claim failed")
		return
	}
	if len(claimed) > 0 {
		tasksClaimed.WithLabelValues().Add(float64(len(claimed)))
		workerQueueDepth.Set(float64(len(s.taskCh)))
	}
	for _, t := range claimed {
		s.dispatch(t, now)
	}
}

// dispatch registers the task on the wheel; the callback enqueues it to the worker
// pool non-blocking so a saturated pool never stalls the wheel.
func (s *Scheduler) dispatch(t *model.Task, now time.Time) {
	delay := t.ExecTime.Sub(now)
	err := s.wheel.Add(delay, t.Id, func() {
		select {
		case s.taskCh <- t:
		default:
			log.Error().Any("task_id", t.Id).Msg("scheduler: worker pool full, dropping task (stays claimed)")
		}
	})
	if err != nil {
		log.Warn().Any("error", err).Any("task_id", t.Id).Msg("scheduler: wheel.Add failed")
	}
}

func (s *Scheduler) workerLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.stopCh:
			// drain remaining tasks before exiting so every enqueued task runs.
			select {
			case t := <-s.taskCh:
				s.executeAndFinalize(t)
				continue
			default:
				return
			}
		case t := <-s.taskCh:
			s.executeAndFinalize(t)
		}
	}
}

// maxErrorMessageLen caps an executor error message stored in an audit row so a
// pathological error cannot inflate the row.
const maxErrorMessageLen = 8 << 10 // 8 KiB

// executeAndFinalize runs the executor and marks the task succeeded on success. A
// panic in the executor is isolated so the worker keeps serving other tasks.
// Execution uses context.Background() so in-flight work finishes during shutdown
// drain (the scan ctx is cancelled by then). Every execution (success + all
// failures) is recorded to the audit log before the state-machine transition.
func (s *Scheduler) executeAndFinalize(t *model.Task) {
	attempt := t.Attempts + 1 // 1-based index of THIS execution
	started := time.Now()
	var execResp *ExecutionResponse // declared early so the panic defer can read it
	defer func() {
		// A panic inside Execute skips the normal record/metrics below, so on
		// panic we record a failure here instead. The state machine is NOT
		// mutated: the task stays claimed and is reclaimed by the reaper.
		if r := recover(); r != nil {
			finished := time.Now()
			errMsg := truncateErr(fmt.Sprintf("panic: %v", r), maxErrorMessageLen)
			log.Error().Any("panic", r).Any("task_id", t.Id).Msg("scheduler: executor panic")
			s.record(t, attempt, enum.ExecutionFailure, errMsg, started, finished, execResp)
			executionDuration.WithLabelValues().Observe(finished.Sub(started).Seconds())
			executionsTotal.WithLabelValues(string(enum.ExecutionFailure)).Inc()
		}
	}()

	dom := domain.NewFromDO(t)
	var err error
	execResp, err = s.exec.Execute(context.Background(), dom)
	finished := time.Now()

	// Record every execution BEFORE the state-machine transition so the audit row
	// survives a crash between Execute returning and the Mark* call. Best-effort:
	// a record failure is logged and swallowed, never affecting scheduling.
	result := enum.ExecutionSuccess
	errMsg := ""
	if err != nil {
		result = enum.ExecutionFailure
		errMsg = truncateErr(err.Error(), maxErrorMessageLen)
	}
	s.record(t, attempt, result, errMsg, started, finished, execResp)
	executionDuration.WithLabelValues().Observe(finished.Sub(started).Seconds())
	executionsTotal.WithLabelValues(string(result)).Inc()

	if err != nil {
		var nr *NonRetryableError
		if errors.As(err, &nr) {
			log.Info().Any("task_id", t.Id).Any("error", err.Error()).Msg("scheduler: execute failed, non-retryable -> dead")
			if derr := s.repo.MarkDead(context.Background(), t.Id, err.Error()); derr != nil {
				log.Error().Any("error", derr).Any("task_id", t.Id).Msg("scheduler: mark dead failed")
			}
		} else {
			log.Info().Any("task_id", t.Id).Any("attempts", t.Attempts).Any("error", err.Error()).Msg("scheduler: execute failed, will retry/dead")
			next := time.Now().Add(exponentialBackoff(t.Attempts, s.cfg.BackoffBase, s.cfg.BackoffMaxInterval))
			if _, ferr := s.repo.MarkFailure(context.Background(), t.Id, err.Error(), next); ferr != nil {
				log.Error().Any("error", ferr).Any("task_id", t.Id).Msg("scheduler: mark failure failed")
			}
		}
		return
	}
	log.Info().Any("task_id", t.Id).Msg("scheduler: execute succeeded")
	if err := s.repo.MarkSucceeded(context.Background(), t.Id); err != nil {
		log.Error().Any("error", err).Any("task_id", t.Id).Msg("scheduler: mark succeeded failed")
	}
}

// record appends one execution-audit row via the recorder. Best-effort: any error
// is logged and swallowed so auditing never affects the task state machine. A nil
// recorder (the default) is a no-op, letting tests inject a taskClaimer without
// implementing the recorder.
func (s *Scheduler) record(t *model.Task, attempt int, result enum.Result, errMsg string, started, finished time.Time, execResp *ExecutionResponse) {
	if s.recorder == nil {
		return
	}
	rec := &model.TaskRecord{
		TaskId:       t.Id,
		Attempt:      attempt,
		Result:       result,
		Protocol:     t.Protocol,
		InstanceId:   s.cfg.InstanceID,
		ErrorMessage: errMsg,
		StartedAt:    started,
		FinishedAt:   finished,
		DurationMs:   finished.Sub(started).Milliseconds(),
	}
	if execResp != nil {
		// Best-effort: store the upstream response as JSON. Left nil when the
		// executor returned none (connect failure, panic, noop).
		if data, merr := json.Marshal(execResp); merr == nil {
			rec.Response = datatypes.JSON(data)
		}
	}
	if rerr := s.recorder.Record(context.Background(), rec); rerr != nil {
		log.Error().
			Any("error", rerr).
			Any("task_id", t.Id).
			Any("attempt", attempt).
			Msg("scheduler: record execution failed (best-effort, ignored)")
	}
}

// truncateErr caps s at maxLen runes (with a trailing ellipsis) so a pathological
// executor error cannot inflate an audit row. Operates on runes to avoid breaking
// multi-byte characters.
func truncateErr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen]) + "..."
}
