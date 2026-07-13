package cron

import (
	"testing"
	"time"
)

func TestValidate_Accepts(t *testing.T) {
	for _, expr := range []string{
		"0 9 * * *",
		"*/5 * * * *",
		"@every 5m",
		"@daily",
	} {
		if err := Validate(expr); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", expr, err)
		}
	}
}

func TestValidate_Rejects(t *testing.T) {
	for _, expr := range []string{
		"bad spec",
		"TZ=America/New_York 0 9 * * *",
		"CRON_TZ=UTC 0 9 * * *",
	} {
		if err := Validate(expr); err == nil {
			t.Errorf("Validate(%q) = nil, want error", expr)
		}
	}
}

func TestNext_StandardFields(t *testing.T) {
	utc, _ := time.LoadLocation("UTC")
	// 08:59:59 -> next 09:00:00 same day.
	from := time.Date(2026, 7, 11, 8, 59, 59, 0, utc)
	got, err := Next("0 9 * * *", from, utc)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	want := time.Date(2026, 7, 11, 9, 0, 0, 0, utc)
	if !got.Equal(want) {
		t.Errorf("Next = %v, want %v", got, want)
	}
	// Next is strictly after the input: exactly 09:00:00 -> next day.
	from = time.Date(2026, 7, 11, 9, 0, 0, 0, utc)
	got, _ = Next("0 9 * * *", from, utc)
	want = time.Date(2026, 7, 12, 9, 0, 0, 0, utc)
	if !got.Equal(want) {
		t.Errorf("Next = %v, want %v (next day)", got, want)
	}
}

func TestNext_EveryDescriptor(t *testing.T) {
	utc, _ := time.LoadLocation("UTC")
	from := time.Date(2026, 7, 11, 12, 0, 0, 0, utc)
	got, err := Next("@every 5m", from, utc)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	want := from.Add(5 * time.Minute)
	if !got.Equal(want) {
		t.Errorf("Next = %v, want %v", got, want)
	}
}

func TestNext_Timezone(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	// 2026-07-11 is EDT (UTC-4). A 0 9 * * * spec in NY yields 09:00 ET.
	from := time.Date(2026, 7, 11, 8, 59, 59, 0, ny)
	got, err := Next("0 9 * * *", from, ny)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	want := time.Date(2026, 7, 11, 9, 0, 0, 0, ny)
	if !got.Equal(want) {
		t.Errorf("Next = %v, want %v (09:00 NY)", got, want)
	}
}

func TestNext_UnsatisfiableReturnsZero(t *testing.T) {
	utc, _ := time.LoadLocation("UTC")
	from := time.Date(2026, 7, 11, 0, 0, 0, 0, utc)
	got, err := Next("0 9 31 2 *", from, utc)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("Next = %v, want zero (unsatisfiable spec)", got)
	}
}

func TestNext_CachesSchedule(t *testing.T) {
	cache.Range(func(k, v any) bool { cache.Delete(k); return true }) // start clean
	utc, _ := time.LoadLocation("UTC")
	from := time.Date(2026, 7, 11, 0, 0, 0, 0, utc)

	_, _ = Next("@every 1m", from, utc)
	v1, ok := cache.Load("@every 1m")
	if !ok {
		t.Fatal("schedule not cached after first Next")
	}
	_, _ = Next("@every 1m", from, utc)
	v2, _ := cache.Load("@every 1m")
	if v1 != v2 {
		t.Error("schedule pointer changed between calls (cache miss)")
	}
}
