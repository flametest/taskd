package scheduler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/internal/domain"
	"github.com/flametest/taskd/internal/infra/model"
	"github.com/flametest/taskd/pkg/timingwheel"
	"github.com/flametest/vita/vgorm"
	log "github.com/flametest/vita/vlog"
)

func TestMain(m *testing.M) {
	// Initialize the global vlog so scheduler log calls do not dereference a nil
	// Event (which would crash worker goroutines on the error/panic paths).
	log.InitLogger(log.ZerologType, "scheduler-test", log.InfoLevel)
	os.Exit(m.Run())
}

// --- fakes ---

// fakeRepo is an in-memory taskClaimer. Its default Claim flips in-window scheduled
// rows to claimed; claimFn overrides that for specialized tests.
type fakeRepo struct {
	mu        sync.Mutex
	tasks     map[string]*model.Task
	succCalls int
	claimErr  error
	succErr   error
	claimFn   func(now time.Time, lookahead time.Duration, batch int, lease time.Duration) ([]*model.Task, error)
}

func newFakeRepo(tasks ...*model.Task) *fakeRepo {
	m := make(map[string]*model.Task, len(tasks))
	for _, t := range tasks {
		m[t.Id] = t
	}
	return &fakeRepo{tasks: m}
}

func (f *fakeRepo) Claim(ctx context.Context, now time.Time, lookahead time.Duration, batchSize int, lease time.Duration) ([]*model.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	if f.claimFn != nil {
		return f.claimFn(now, lookahead, batchSize, lease)
	}
	cutoff := now.Add(lookahead)
	leaseUntil := now.Add(lease)
	var out []*model.Task
	for _, t := range f.tasks {
		if t.Status == enum.TaskStatusScheduled && !t.ExecTime.After(cutoff) {
			t.Status = enum.TaskStatusClaimed
			t.LockedUntil = &leaseUntil
			out = append(out, t)
			if len(out) >= batchSize {
				break
			}
		}
	}
	return out, nil
}

func (f *fakeRepo) MarkSucceeded(ctx context.Context, taskId string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.succErr != nil {
		return f.succErr
	}
	t, ok := f.tasks[taskId]
	if !ok || t.Status != enum.TaskStatusClaimed {
		return errors.New("not claimed")
	}
	t.Status = enum.TaskStatusSucceeded
	f.succCalls++
	return nil
}

func (f *fakeRepo) succCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.succCalls
}

// MarkFailure mirrors the repo SQL: increment attempts, store last_error, then
// either re-schedule (attempts <= max_retries) or mark dead.
func (f *fakeRepo) MarkFailure(ctx context.Context, taskId string, lastError string, nextExecTime time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tasks[taskId]
	if !ok || t.Status != enum.TaskStatusClaimed {
		return 0, nil
	}
	t.Attempts++
	t.LastError = lastError
	t.LockedUntil = nil
	if t.Attempts > t.MaxRetries {
		t.Status = enum.TaskStatusDead
	} else {
		t.Status = enum.TaskStatusScheduled
		t.ExecTime = nextExecTime
	}
	return 1, nil
}

// ReclaimOrphans mirrors the repo SQL: reset claimed+expired-lease rows to scheduled.
func (f *fakeRepo) ReclaimOrphans(ctx context.Context, now time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var n int64
	for _, t := range f.tasks {
		if t.Status == enum.TaskStatusClaimed && t.LockedUntil != nil && t.LockedUntil.Before(now) {
			t.Status = enum.TaskStatusScheduled
			t.LockedUntil = nil
			n++
		}
	}
	return n, nil
}

// MarkDead mirrors the repo SQL: mark a claimed task dead without retrying.
func (f *fakeRepo) MarkDead(ctx context.Context, taskId string, lastError string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tasks[taskId]
	if !ok || t.Status != enum.TaskStatusClaimed {
		return errors.New("not claimed")
	}
	t.Attempts++
	t.Status = enum.TaskStatusDead
	t.LastError = lastError
	t.LockedUntil = nil
	return nil
}

// recordingExecutor records fired task ids on a buffered channel.
type recordingExecutor struct {
	ch    chan string
	err   error
	delay time.Duration
}

func newRecordingExecutor() *recordingExecutor {
	return &recordingExecutor{ch: make(chan string, 1024)}
}

func (r *recordingExecutor) Execute(ctx context.Context, task *domain.Task) error {
	if r.delay > 0 {
		time.Sleep(r.delay)
	}
	r.ch <- task.Id
	return r.err
}

func (r *recordingExecutor) count() int { return len(r.ch) }

// flakyExecutor fails the first failUntil executions, then succeeds. It records
// every call on ch so waitExec can count total executions across retries.
type flakyExecutor struct {
	ch        chan string
	failUntil int
	mu        sync.Mutex
	calls     int
}

func newFlakyExecutor(failUntil int) *flakyExecutor {
	return &flakyExecutor{ch: make(chan string, 1024), failUntil: failUntil}
}

func (f *flakyExecutor) Execute(ctx context.Context, task *domain.Task) error {
	f.mu.Lock()
	f.calls++
	n := f.calls
	f.mu.Unlock()
	f.ch <- task.Id
	if n <= f.failUntil {
		return errors.New("flaky")
	}
	return nil
}

func (f *flakyExecutor) count() int { return len(f.ch) }

// panicExecutor always panics, to exercise worker panic isolation.
type panicExecutor struct{}

func (panicExecutor) Execute(ctx context.Context, task *domain.Task) error { panic("boom") }

// blockingExecutor blocks inside Execute until release is signaled; used to test
// graceful drain of in-flight work.
type blockingExecutor struct {
	started chan struct{}
	release chan struct{}
}

func (b *blockingExecutor) Execute(ctx context.Context, task *domain.Task) error {
	b.started <- struct{}{}
	<-b.release
	return nil
}

// --- helpers ---

func testCfg() SchedulerConfig {
	cfg := ResolveSchedulerConfig(SchedulerConfig{})
	cfg.ScanInterval = 5 * time.Millisecond
	cfg.TickInterval = 5 * time.Millisecond
	cfg.LookaheadWindow = 100 * time.Millisecond
	cfg.WorkerConcurrency = 2
	cfg.BatchSize = 16
	return cfg
}

func newScheduledTask(id string, execTime time.Time) *model.Task {
	return &model.Task{
		BasePostgres: vgorm.BasePostgres{Id: id},
		RefId:        fmt.Sprintf("task-%s", id),
		Name:         "test",
		Protocol:     enum.ProtocolHTTP,
		Address:      "http://x",
		ExecTime:     execTime,
		Status:       enum.TaskStatusScheduled,
		MaxRetries:   3,
	}
}

// startScheduler builds a real system-clock wheel + scheduler, starts it, and
// arranges cleanup. Returns the scheduler and a cancel func for the scan ctx.
func startScheduler(t *testing.T, cfg SchedulerConfig, repo taskClaimer, exec Executor) (*Scheduler, context.CancelFunc) {
	t.Helper()
	wheel := timingwheel.New(
		timingwheel.WithTickInterval(cfg.TickInterval),
		timingwheel.WithSlotsPerLevel(cfg.SlotsPerLevel),
		timingwheel.WithMaxLevels(cfg.MaxLevels),
	)
	s := NewScheduler(cfg, repo, wheel, exec)
	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)
	t.Cleanup(func() {
		cancel()
		s.Stop()
	})
	return s, cancel
}

// counter is satisfied by any executor that reports how many times it has run.
type counter interface{ count() int }

func waitExec(t *testing.T, c counter, n int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if c.count() >= n {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("expected %d executions within %v, got %d", n, timeout, c.count())
}

// waitStatus polls the fake repo until taskId reaches the wanted status.
func waitStatus(t *testing.T, repo *fakeRepo, taskId string, want enum.Status, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		repo.mu.Lock()
		task, ok := repo.tasks[taskId]
		var st enum.Status = enum.TaskStatusScheduled
		if ok {
			st = task.Status
		}
		repo.mu.Unlock()
		if st == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("task %s did not reach %s within %v", taskId, want, timeout)
}

// --- tests ---

func TestScheduler_StartStop_Idempotent(t *testing.T) {
	repo := newFakeRepo()
	rec := newRecordingExecutor()
	s, cancel := startScheduler(t, testCfg(), repo, rec)
	s.Start(context.Background()) // second Start is a no-op
	cancel()
	s.Stop()
	s.Stop() // second Stop is a no-op
}

func TestScheduler_StopBeforeStart(t *testing.T) {
	wheel := timingwheel.New(timingwheel.WithTickInterval(5 * time.Millisecond))
	s := NewScheduler(testCfg(), newFakeRepo(), wheel, newRecordingExecutor())
	s.Stop() // must not panic or block
}

func TestScheduler_ClaimedTask_FiresAndSucceeds(t *testing.T) {
	task := newScheduledTask("1", time.Now().Add(20*time.Millisecond))
	repo := newFakeRepo(task)
	rec := newRecordingExecutor()
	s, cancel := startScheduler(t, testCfg(), repo, rec)
	waitExec(t, rec, 1, 500*time.Millisecond)
	cancel()
	s.Stop()
	if task.Status != enum.TaskStatusSucceeded {
		t.Errorf("status = %s, want succeeded", task.Status)
	}
}

func TestScheduler_PastTask_FiresImmediately(t *testing.T) {
	task := newScheduledTask("1", time.Now().Add(-1*time.Second))
	repo := newFakeRepo(task)
	rec := newRecordingExecutor()
	s, cancel := startScheduler(t, testCfg(), repo, rec)
	waitExec(t, rec, 1, 200*time.Millisecond)
	cancel()
	s.Stop()
}

func TestScheduler_FutureTask_NotFiredEarly(t *testing.T) {
	task := newScheduledTask("1", time.Now().Add(1*time.Second))
	repo := newFakeRepo(task)
	rec := newRecordingExecutor()
	cfg := testCfg()
	cfg.LookaheadWindow = 0 // task falls outside the claim window
	s, cancel := startScheduler(t, cfg, repo, rec)
	time.Sleep(150 * time.Millisecond)
	if rec.count() != 0 {
		t.Fatalf("future task executed early: %d", rec.count())
	}
	cancel()
	s.Stop()
	if task.Status != enum.TaskStatusScheduled {
		t.Errorf("status = %s, want scheduled (outside lookahead window)", task.Status)
	}
}

func TestScheduler_LookaheadPreClaims(t *testing.T) {
	// exec_time is 50ms out, but LookaheadWindow (100ms) covers it -> claimed on
	// the first scan, then fires ~50ms later.
	task := newScheduledTask("1", time.Now().Add(50*time.Millisecond))
	repo := newFakeRepo(task)
	rec := newRecordingExecutor()
	s, cancel := startScheduler(t, testCfg(), repo, rec)
	waitExec(t, rec, 1, 300*time.Millisecond)
	cancel()
	s.Stop()
}

func TestScheduler_WorkerPool_Parallelism(t *testing.T) {
	now := time.Now()
	tasks := []*model.Task{
		newScheduledTask("1", now),
		newScheduledTask("2", now),
		newScheduledTask("3", now),
		newScheduledTask("4", now),
	}
	repo := newFakeRepo(tasks...)
	rec := newRecordingExecutor()
	rec.delay = 30 * time.Millisecond
	cfg := testCfg()
	cfg.WorkerConcurrency = 4
	start := time.Now()
	s, cancel := startScheduler(t, cfg, repo, rec)
	waitExec(t, rec, 4, 500*time.Millisecond)
	elapsed := time.Since(start)
	cancel()
	s.Stop()
	// 4 tasks * 30ms on 4 workers should run in parallel (~30ms), well under the
	// ~120ms serial bound. Allow headroom for scheduler/wheel startup.
	if elapsed >= 100*time.Millisecond {
		t.Errorf("not parallel: elapsed = %v (want < 100ms for 4x30ms on 4 workers)", elapsed)
	}
}

func TestScheduler_ExecutorError_RetriesThenSucceeds(t *testing.T) {
	task := newScheduledTask("1", time.Now())
	task.MaxRetries = 3
	repo := newFakeRepo(task)
	flaky := newFlakyExecutor(2) // first 2 fail, 3rd succeeds
	cfg := testCfg()
	cfg.BackoffBase = 5 * time.Millisecond
	cfg.BackoffMaxInterval = 20 * time.Millisecond
	s, cancel := startScheduler(t, cfg, repo, flaky)
	waitExec(t, flaky, 3, 2*time.Second) // 2 failures (re-scheduled) + 1 success
	cancel()
	s.Stop()
	if task.Status != enum.TaskStatusSucceeded {
		t.Errorf("status = %s, want succeeded after retries", task.Status)
	}
	if task.Attempts != 2 {
		t.Errorf("attempts = %d, want 2", task.Attempts)
	}
}

func TestScheduler_ExecutorError_RetriesToDead(t *testing.T) {
	task := newScheduledTask("1", time.Now())
	task.MaxRetries = 2
	repo := newFakeRepo(task)
	rec := newRecordingExecutor()
	rec.err = errors.New("boom") // always fails
	cfg := testCfg()
	cfg.BackoffBase = 5 * time.Millisecond
	cfg.BackoffMaxInterval = 20 * time.Millisecond
	s, cancel := startScheduler(t, cfg, repo, rec)
	// attempts: 0->1 (retry), 1->2 (retry), 2->3 (>2 -> dead): 3 executions total.
	waitExec(t, rec, 3, 2*time.Second)
	waitStatus(t, repo, task.Id, enum.TaskStatusDead, 1*time.Second)
	cancel()
	s.Stop()
	if task.Status != enum.TaskStatusDead {
		t.Errorf("status = %s, want dead", task.Status)
	}
	if task.Attempts != 3 {
		t.Errorf("attempts = %d, want 3", task.Attempts)
	}
	if task.LastError == "" {
		t.Error("last_error is empty, want the failure reason")
	}
}

func TestScheduler_Reaper_ReclaimsOrphan(t *testing.T) {
	// A claimed task whose lease already expired; the reaper should reset it to
	// 'scheduled', after which the scan loop claims and executes it to 'succeeded'.
	past := time.Now().Add(-1 * time.Minute)
	task := &model.Task{
		BasePostgres: vgorm.BasePostgres{Id: "1"},
		RefId:        "orphan-1",
		Name:         "orphan",
		Protocol:     enum.ProtocolHTTP,
		Address:      "http://x",
		ExecTime:     time.Now().Add(-1 * time.Minute),
		Status:       enum.TaskStatusClaimed,
		MaxRetries:   3,
		LockedUntil:  &past,
	}
	repo := newFakeRepo(task)
	rec := newRecordingExecutor()
	cfg := testCfg()
	cfg.ReaperInterval = 10 * time.Millisecond
	s, cancel := startScheduler(t, cfg, repo, rec)
	waitStatus(t, repo, task.Id, enum.TaskStatusSucceeded, 1*time.Second)
	cancel()
	s.Stop()
}

// nonRetryableExecutor always returns a NonRetryableError, exercising the
// scheduler's dead-without-retry path (e.g. HTTP 4xx).
type nonRetryableExecutor struct{}

func (nonRetryableExecutor) Execute(ctx context.Context, task *domain.Task) error {
	return NewNonRetryableError(errors.New("bad request"))
}

func TestScheduler_NonRetryableGoesDead(t *testing.T) {
	task := newScheduledTask("1", time.Now())
	task.MaxRetries = 3
	repo := newFakeRepo(task)
	cfg := testCfg()
	s, cancel := startScheduler(t, cfg, repo, nonRetryableExecutor{})
	waitStatus(t, repo, task.Id, enum.TaskStatusDead, 2*time.Second)
	cancel()
	s.Stop()
	if task.Attempts != 1 {
		t.Errorf("attempts = %d, want 1 (non-retryable, single attempt)", task.Attempts)
	}
	if task.LastError == "" {
		t.Error("last_error is empty, want the failure reason")
	}
}

func TestScheduler_ExecutorPanic_WorkerSurvives(t *testing.T) {
	task1 := newScheduledTask("1", time.Now().Add(10*time.Millisecond))
	task2 := newScheduledTask("2", time.Now().Add(10*time.Millisecond))
	repo := newFakeRepo(task1, task2)
	s, cancel := startScheduler(t, testCfg(), repo, panicExecutor{})
	time.Sleep(250 * time.Millisecond) // let both fire and panic-recover
	cancel()
	s.Stop()
	if task1.Status != enum.TaskStatusClaimed || task2.Status != enum.TaskStatusClaimed {
		t.Errorf("expected both claimed after panic (no MarkSucceeded); got %s, %s", task1.Status, task2.Status)
	}
}

func TestScheduler_DuplicateWheelKey_NoDoubleFireBeforeExpiry(t *testing.T) {
	// claimFn returns the same task on every scan without flipping status, so the
	// second and later wheel.Add calls for the same key are rejected with
	// ErrKeyExists. The task must not fire multiple times before its exec_time.
	task := newScheduledTask("1", time.Now().Add(500*time.Millisecond))
	repo := newFakeRepo(task)
	repo.claimFn = func(now time.Time, lookahead time.Duration, batch int, lease time.Duration) ([]*model.Task, error) {
		return []*model.Task{task}, nil
	}
	rec := newRecordingExecutor()
	s, cancel := startScheduler(t, testCfg(), repo, rec)
	// many scans re-Add the same key; only the first Add registers, the rest are
	// rejected. exec_time is 500ms out, so nothing should have fired yet.
	time.Sleep(200 * time.Millisecond)
	if rec.count() != 0 {
		t.Errorf("expected 0 executions before exec_time, got %d", rec.count())
	}
	cancel()
	s.Stop()
}

func TestScheduler_GracefulShutdown_DrainsInflight(t *testing.T) {
	task := newScheduledTask("1", time.Now().Add(10*time.Millisecond))
	repo := newFakeRepo(task)
	exec := &blockingExecutor{
		started: make(chan struct{}, 1),
		release: make(chan struct{}, 1),
	}
	cfg := testCfg()
	wheel := timingwheel.New(
		timingwheel.WithTickInterval(cfg.TickInterval),
		timingwheel.WithSlotsPerLevel(cfg.SlotsPerLevel),
		timingwheel.WithMaxLevels(cfg.MaxLevels),
	)
	s := NewScheduler(cfg, repo, wheel, exec)
	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	<-exec.started // task fired, executor is now blocked inside Execute
	cancel()       // stop the scan loop

	done := make(chan struct{})
	go func() {
		exec.release <- struct{}{} // let the in-flight execution finish
		s.Stop()                   // drains the worker, then returns
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not return within 2s (drain stuck)")
	}
	if task.Status != enum.TaskStatusSucceeded {
		t.Errorf("status = %s, want succeeded (in-flight execution drained)", task.Status)
	}
}

func TestResolveSchedulerConfig_DefaultsAndInstanceID(t *testing.T) {
	cfg := ResolveSchedulerConfig(SchedulerConfig{})
	if cfg.ScanInterval != 1*time.Second {
		t.Errorf("ScanInterval = %v, want 1s", cfg.ScanInterval)
	}
	if cfg.BatchSize != 100 {
		t.Errorf("BatchSize = %d, want 100", cfg.BatchSize)
	}
	if cfg.WorkerConcurrency != 8 {
		t.Errorf("WorkerConcurrency = %d, want 8", cfg.WorkerConcurrency)
	}
	if cfg.InstanceID == "" {
		t.Error("InstanceID is empty, want hostname-pid default")
	}

	cfg2 := ResolveSchedulerConfig(SchedulerConfig{InstanceID: "explicit-id"})
	if cfg2.InstanceID != "explicit-id" {
		t.Errorf("InstanceID = %q, want preserved explicit-id", cfg2.InstanceID)
	}
}
