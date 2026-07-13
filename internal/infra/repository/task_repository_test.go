package repository

import (
	"context"
	"testing"
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/internal/infra/model"
	"github.com/flametest/vita/vgorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// setupSQLiteTaskDB opens an in-memory SQLite DB with a SQLite-adapted task
// table (PG-only DDL is not exercised here — see taskd-test-gotchas).
func setupSQLiteTaskDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Exec("DROP TABLE IF EXISTS task").Error })
	ddls := []string{
		`CREATE TABLE task (
			id TEXT PRIMARY KEY,
			version INTEGER NOT NULL DEFAULT 0,
			name TEXT NOT NULL,
			ref_id TEXT NOT NULL,
			protocol TEXT NOT NULL,
			address TEXT NOT NULL,
			params BLOB,
			exec_time DATETIME NOT NULL,
			status TEXT NOT NULL,
			attempts INTEGER NOT NULL DEFAULT 0,
			max_retries INTEGER NOT NULL,
			last_error TEXT,
			locked_until DATETIME,
			cron TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at DATETIME
		)`,
	}
	for _, ddl := range ddls {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("exec ddl: %v", err)
		}
	}
	return db
}

func newTask(id string, status enum.Status) *model.Task {
	return &model.Task{
		BasePostgres: vgorm.BasePostgres{Id: id},
		Name:         "t",
		RefId:        id,
		Protocol:     enum.ProtocolHTTP,
		Address:      "http://x",
		ExecTime:     time.Now(),
		Status:       status,
		MaxRetries:   3,
	}
}

func TestTaskRepository_Reactivate(t *testing.T) {
	db := setupSQLiteTaskDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	dead := newTask("t-dead", enum.TaskStatusDead)
	dead.Attempts = 5
	dead.LastError = "boom"
	if err := db.Create(dead).Error; err != nil {
		t.Fatalf("seed dead: %v", err)
	}
	scheduled := newTask("t-scheduled", enum.TaskStatusScheduled)
	if err := db.Create(scheduled).Error; err != nil {
		t.Fatalf("seed scheduled: %v", err)
	}

	if err := repo.Reactivate(ctx, "t-dead", time.Now()); err != nil {
		t.Fatalf("Reactivate dead: %v", err)
	}
	var got model.Task
	if err := db.First(&got, "id = ?", "t-dead").Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Status != enum.TaskStatusScheduled {
		t.Errorf("status = %s, want scheduled", got.Status)
	}
	if got.Attempts != 0 {
		t.Errorf("attempts = %d, want 0 (reset on reactivate)", got.Attempts)
	}
	if got.LastError != "" {
		t.Errorf("last_error = %q, want empty (cleared on reactivate)", got.LastError)
	}

	// Reactivating a non-dead task must fail.
	if err := repo.Reactivate(ctx, "t-scheduled", time.Now()); err == nil {
		t.Error("Reactivate scheduled: expected conflict error, got nil")
	}
}

func TestTaskRepository_Cancel(t *testing.T) {
	db := setupSQLiteTaskDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	scheduled := newTask("t-scheduled", enum.TaskStatusScheduled)
	if err := db.Create(scheduled).Error; err != nil {
		t.Fatalf("seed scheduled: %v", err)
	}
	claimed := newTask("t-claimed", enum.TaskStatusClaimed)
	if err := db.Create(claimed).Error; err != nil {
		t.Fatalf("seed claimed: %v", err)
	}

	if err := repo.Cancel(ctx, "t-scheduled"); err != nil {
		t.Fatalf("Cancel scheduled: %v", err)
	}
	var got model.Task
	if err := db.First(&got, "id = ?", "t-scheduled").Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Status != enum.TaskStatusCanceled {
		t.Errorf("status = %s, want canceled", got.Status)
	}

	// Canceling a claimed (executing) task must fail.
	if err := repo.Cancel(ctx, "t-claimed"); err == nil {
		t.Error("Cancel claimed: expected conflict error, got nil")
	}
}

func TestTaskRepository_Reschedule(t *testing.T) {
	db := setupSQLiteTaskDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	claimed := newTask("t-claimed", enum.TaskStatusClaimed)
	claimed.Attempts = 2
	claimed.LastError = "boom"
	if err := db.Create(claimed).Error; err != nil {
		t.Fatalf("seed claimed: %v", err)
	}
	scheduled := newTask("t-scheduled", enum.TaskStatusScheduled)
	if err := db.Create(scheduled).Error; err != nil {
		t.Fatalf("seed scheduled: %v", err)
	}

	next := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	if err := repo.Reschedule(ctx, "t-claimed", next); err != nil {
		t.Fatalf("Reschedule claimed: %v", err)
	}
	var got model.Task
	if err := db.First(&got, "id = ?", "t-claimed").Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Status != enum.TaskStatusScheduled {
		t.Errorf("status = %s, want scheduled", got.Status)
	}
	if got.Attempts != 0 {
		t.Errorf("attempts = %d, want 0 (reset on reschedule)", got.Attempts)
	}
	if got.LastError != "" {
		t.Errorf("last_error = %q, want empty (cleared on reschedule)", got.LastError)
	}
	if !got.ExecTime.Equal(next) {
		t.Errorf("exec_time = %v, want %v", got.ExecTime, next)
	}
	if got.LockedUntil != nil {
		t.Errorf("locked_until = %v, want nil (lease cleared)", got.LockedUntil)
	}

	// Rescheduling a non-claimed task must fail.
	if err := repo.Reschedule(ctx, "t-scheduled", next); err == nil {
		t.Error("Reschedule scheduled: expected conflict error, got nil")
	}
}
