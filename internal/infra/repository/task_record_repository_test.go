package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/flametest/taskd/internal/constant/enum"
	"github.com/flametest/taskd/internal/infra/model"
	log "github.com/flametest/vita/vlog"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// TestMain initializes vlog defensively; the repo under test does not log, but
// imported helpers (e.g. verrors) may on error paths.
func TestMain(m *testing.M) {
	log.InitLogger(log.ZerologType, "repo-test", log.InfoLevel)
	os.Exit(m.Run())
}

// setupSQLiteDB opens an in-memory SQLite DB and creates a SQLite-adapted
// task_record table. PG-specific DDL (gen_random_uuid(), TIMESTAMPTZ) is NOT
// exercised here — only GORM mapping and query logic. PostgreSQL behavior is
// validated manually, consistent with the rest of this package (see the
// SKIP LOCKED note in task_repsitory.go).
func setupSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		// Match vgorm.NewDB's naming so model.TaskRecord maps to table
		// "task_record" (singular), consistent with migration/init.sql.
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec("DROP TABLE IF EXISTS task_record").Error
	})
	ddls := []string{
		`CREATE TABLE task_record (
			id            TEXT PRIMARY KEY,
			task_id       TEXT NOT NULL,
			attempt       INTEGER NOT NULL,
			result        TEXT NOT NULL,
			protocol      TEXT NOT NULL,
			instance_id   TEXT NOT NULL,
			error_message TEXT,
			started_at    DATETIME NOT NULL,
			finished_at   DATETIME NOT NULL,
			duration_ms   INTEGER NOT NULL,
			response      BLOB,
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX idx_task_record_task_id_created_at ON task_record (task_id, created_at DESC)`,
	}
	for _, ddl := range ddls {
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("exec ddl: %v", err)
		}
	}
	return db
}

// newTestRecord builds a TaskRecord with a deterministic id and a caller-supplied
// created_at (so ordering tests are independent of sub-second clock resolution).
func newTestRecord(taskId string, attempt int, result enum.Result, createdAt time.Time) *model.TaskRecord {
	return &model.TaskRecord{
		RecordBase: model.RecordBase{Id: fmt.Sprintf("%s-%d", taskId, attempt), CreatedAt: createdAt},
		TaskId:     taskId,
		Attempt:    attempt,
		Result:     result,
		Protocol:   enum.ProtocolHTTP,
		InstanceId: "test-instance",
		StartedAt:  createdAt,
		FinishedAt: createdAt.Add(time.Millisecond),
		DurationMs: 1,
	}
}

func TestTaskRecordRepository_RecordRoundTrip(t *testing.T) {
	db := setupSQLiteDB(t)
	repo := NewTaskRecordRepository(db)
	base := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	r := newTestRecord("t1", 1, enum.ExecutionSuccess, base)
	if err := repo.Record(context.Background(), r); err != nil {
		t.Fatalf("Record: %v", err)
	}

	out, err := repo.ListByTaskId(context.Background(), "t1", 10, 0)
	if err != nil {
		t.Fatalf("ListByTaskId: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}
	got := out[0]
	if got.TaskId != "t1" || got.Attempt != 1 || got.Result != enum.ExecutionSuccess ||
		got.Protocol != enum.ProtocolHTTP || got.InstanceId != "test-instance" ||
		got.DurationMs != 1 || got.ErrorMessage != "" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if len(got.Response) != 0 {
		t.Errorf("Response = %v, want empty this round", got.Response)
	}
}

func TestTaskRecordRepository_ListByTaskId_Ordering(t *testing.T) {
	db := setupSQLiteDB(t)
	repo := NewTaskRecordRepository(db)
	base := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	// Insert out of order; expect newest-first (created_at DESC).
	for _, a := range []int{1, 3, 2} {
		if err := repo.Record(context.Background(), newTestRecord("t1", a, enum.ExecutionFailure, base.Add(time.Duration(a)*time.Second))); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}
	// A different task's records must not leak in.
	if err := repo.Record(context.Background(), newTestRecord("other", 1, enum.ExecutionSuccess, base)); err != nil {
		t.Fatalf("Record: %v", err)
	}

	out, err := repo.ListByTaskId(context.Background(), "t1", 100, 0)
	if err != nil {
		t.Fatalf("ListByTaskId: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("len = %d, want 3", len(out))
	}
	// Newest-first: created_at descending -> attempts 3, 2, 1.
	want := []int{3, 2, 1}
	for i, w := range want {
		if out[i].Attempt != w {
			t.Errorf("out[%d].Attempt = %d, want %d (newest-first)", i, out[i].Attempt, w)
		}
	}
}

func TestTaskRecordRepository_ListByTaskId_LimitOffset(t *testing.T) {
	db := setupSQLiteDB(t)
	repo := NewTaskRecordRepository(db)
	base := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	for i := 1; i <= 5; i++ {
		if err := repo.Record(context.Background(), newTestRecord("t1", i, enum.ExecutionSuccess, base.Add(time.Duration(i)*time.Second))); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	// limit<=0 clamps to the default (100), so all 5 are returned here.
	all, err := repo.ListByTaskId(context.Background(), "t1", 0, 0)
	if err != nil {
		t.Fatalf("ListByTaskId limit=0: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("limit=0 returned %d, want 5 (default clamp)", len(all))
	}

	// limit=2 -> the 2 newest.
	page, err := repo.ListByTaskId(context.Background(), "t1", 2, 0)
	if err != nil {
		t.Fatalf("ListByTaskId limit=2: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("limit=2 returned %d, want 2", len(page))
	}
	if page[0].Attempt != 5 || page[1].Attempt != 4 {
		t.Errorf("first page attempts = %d,%d, want 5,4 (newest-first)", page[0].Attempt, page[1].Attempt)
	}

	// offset=2 skips the 2 newest.
	page2, err := repo.ListByTaskId(context.Background(), "t1", 2, 2)
	if err != nil {
		t.Fatalf("ListByTaskId offset=2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("offset=2 returned %d, want 2", len(page2))
	}
	if page2[0].Attempt != 3 || page2[1].Attempt != 2 {
		t.Errorf("second page attempts = %d,%d, want 3,2", page2[0].Attempt, page2[1].Attempt)
	}
}
