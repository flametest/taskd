package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/flametest/taskd/internal/domain"
	"github.com/flametest/taskd/internal/infra/model"
	"github.com/flametest/taskd/pkg/timingwheel"
	log "github.com/flametest/vita/vlog"
)

// taskClaimer is the narrow slice of repository.TaskRepository the scheduler
// needs, declared locally so tests can inject a fake without mocking the whole
// repository.
type taskClaimer interface {
	Claim(ctx context.Context, now time.Time, lookahead time.Duration, batchSize int, lease time.Duration) ([]*model.Task, error)
	MarkSucceeded(ctx context.Context, taskId string) error
	MarkFailure(ctx context.Context, taskId string, lastError string, nextExecTime time.Time) (int64, error)
	ReclaimOrphans(ctx context.Context, now time.Time) (int64, error)
}

// Scheduler claims due tasks, hands them to a TimingWheel for precise firing, and
// executes fired tasks on a fixed-size worker pool. Round 1 closes the loop
// scheduled -> claimed -> succeeded; the failure/retry path is a later round.
type Scheduler struct {
	cfg   SchedulerConfig
	repo  taskClaimer
	wheel timingwheel.TimingWheel
	exec  Executor

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

// executeAndFinalize runs the executor and marks the task succeeded on success. A
// panic in the executor is isolated so the worker keeps serving other tasks.
// Execution uses context.Background() so in-flight work finishes during shutdown
// drain (the scan ctx is cancelled by then).
func (s *Scheduler) executeAndFinalize(t *model.Task) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Any("panic", r).Any("task_id", t.Id).Msg("scheduler: executor panic")
		}
	}()
	dom := domain.NewFromDO(t)
	if err := s.exec.Execute(context.Background(), dom); err != nil {
		log.Info().Any("task_id", t.Id).Any("attempts", t.Attempts).Any("error", err.Error()).Msg("scheduler: execute failed, will retry/dead")
		next := time.Now().Add(exponentialBackoff(t.Attempts, s.cfg.BackoffBase, s.cfg.BackoffMaxInterval))
		if _, ferr := s.repo.MarkFailure(context.Background(), t.Id, err.Error(), next); ferr != nil {
			log.Error().Any("error", ferr).Any("task_id", t.Id).Msg("scheduler: mark failure failed")
		}
		return
	}
	log.Info().Any("task_id", t.Id).Msg("scheduler: execute succeeded")
	if err := s.repo.MarkSucceeded(context.Background(), t.Id); err != nil {
		log.Error().Any("error", err).Any("task_id", t.Id).Msg("scheduler: mark succeeded failed")
	}
}
