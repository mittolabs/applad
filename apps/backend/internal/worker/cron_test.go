package worker

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/cronx"
	"github.com/mittolabs/applad/internal/db"
)

// mustParse parses a cron expression or fails the test.
func mustParse(t *testing.T, expr string) cronx.Schedule {
	t.Helper()
	s, err := cronx.Parse(expr)
	if err != nil {
		t.Fatalf("parse %q: %v", expr, err)
	}
	return s
}

func nullTime(tm time.Time) sql.NullTime { return sql.NullTime{Time: tm, Valid: true} }

// TestDecideCron_DueFires: a stored next_run_at in the past means the entity is
// due, so the decision is to claim (and then fire) the occurrence.
func TestDecideCron_DueFires(t *testing.T) {
	sched := mustParse(t, "*/5 * * * *") // every 5 minutes
	now := time.Date(2026, 7, 1, 12, 2, 0, 0, time.UTC)
	storedNext := nullTime(now.Add(-1 * time.Minute)) // one minute overdue

	d := decideCron(sched, true, "*/5 * * * *", "*/5 * * * *", storedNext, now)

	if d.outcome != cronClaim {
		t.Fatalf("due entry should claim, got outcome %v", d.outcome)
	}
	if !d.nextRun.After(now) {
		t.Fatalf("next run should resync to the future, got %v (now %v)", d.nextRun, now)
	}
	if d.nextRun != sched.Next(now) {
		t.Fatalf("next run should be the next occurrence after now: got %v want %v", d.nextRun, sched.Next(now))
	}
}

// TestDecideCron_NotDue: a stored next_run_at in the future means nothing fires.
func TestDecideCron_NotDue(t *testing.T) {
	sched := mustParse(t, "*/5 * * * *")
	now := time.Date(2026, 7, 1, 12, 2, 0, 0, time.UTC)
	storedNext := nullTime(now.Add(3 * time.Minute)) // still in the future

	d := decideCron(sched, true, "*/5 * * * *", "*/5 * * * *", storedNext, now)

	if d.outcome != cronNotDue {
		t.Fatalf("future entry should not fire, got outcome %v", d.outcome)
	}
}

// TestDecideCron_NewSchedule: with no stored row the schedule is seeded from now
// (init, not fire) so it never runs immediately for a window nobody asked for.
func TestDecideCron_NewSchedule(t *testing.T) {
	sched := mustParse(t, "0 9 * * *")
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	d := decideCron(sched, false, "", "0 9 * * *", sql.NullTime{}, now)

	if d.outcome != cronInit {
		t.Fatalf("new schedule should init, got outcome %v", d.outcome)
	}
	if d.nextRun != sched.Next(now) {
		t.Fatalf("init next run mismatch: got %v want %v", d.nextRun, sched.Next(now))
	}
}

// TestDecideCron_ExprChanged: a changed expression re-seeds rather than firing
// against the old schedule's stored next_run.
func TestDecideCron_ExprChanged(t *testing.T) {
	sched := mustParse(t, "0 10 * * *")
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	// storedNext is in the past under the old expression — but the expression
	// changed, so it must not be treated as due.
	storedNext := nullTime(now.Add(-2 * time.Hour))

	d := decideCron(sched, true, "0 9 * * *", "0 10 * * *", storedNext, now)

	if d.outcome != cronInit {
		t.Fatalf("changed expression should init, got outcome %v", d.outcome)
	}
}

// TestDecideCron_NoStoredNext: a row that exists but has no next_run_at yet is
// seeded, still without firing.
func TestDecideCron_NoStoredNext(t *testing.T) {
	sched := mustParse(t, "0 9 * * *")
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	d := decideCron(sched, true, "0 9 * * *", "0 9 * * *", sql.NullTime{}, now)

	if d.outcome != cronInit {
		t.Fatalf("row without next_run should init, got outcome %v", d.outcome)
	}
}

// TestDecideCron_BacklogFiresOnceAndResyncs: a daily job whose stored next_run
// is a week in the past fires exactly once and resyncs its next_run to the next
// future occurrence rather than replaying every missed day.
func TestDecideCron_BacklogFiresOnceAndResyncs(t *testing.T) {
	sched := mustParse(t, "0 0 * * *") // daily at midnight UTC
	now := time.Date(2026, 7, 8, 0, 30, 0, 0, time.UTC)
	storedNext := nullTime(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)) // 7 days ago

	d := decideCron(sched, true, "0 0 * * *", "0 0 * * *", storedNext, now)

	if d.outcome != cronClaim {
		t.Fatalf("backlogged entry should claim, got outcome %v", d.outcome)
	}
	// Occurrences elapsed in (storedNext, now]: Jul 2..Jul 8 midnights = 7.
	if d.missed != 7 {
		t.Fatalf("expected 7 missed occurrences, got %d", d.missed)
	}
	// Fire once: the decision yields a single next run, not one per missed day.
	if d.nextRun != sched.Next(now) {
		t.Fatalf("next run should resync to next occurrence after now: got %v want %v", d.nextRun, sched.Next(now))
	}
	if !d.nextRun.After(now) {
		t.Fatalf("resynced next run must be in the future, got %v", d.nextRun)
	}
	// The resynced next run is one occurrence ahead (Jul 9 midnight), proving the
	// backlog was collapsed rather than left pointing at a past time.
	if want := time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC); d.nextRun != want {
		t.Fatalf("expected resync to %v, got %v", want, d.nextRun)
	}
}

// TestDecideCron_BacklogCappedAt1000 confirms the missed-run counter cannot run
// away on a very old, very frequent schedule.
func TestDecideCron_BacklogCappedAt1000(t *testing.T) {
	sched := mustParse(t, "* * * * *") // every minute
	now := time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC)
	storedNext := nullTime(now.Add(-365 * 24 * time.Hour)) // a year of minutes

	d := decideCron(sched, true, "* * * * *", "* * * * *", storedNext, now)

	if d.outcome != cronClaim {
		t.Fatalf("expected claim, got %v", d.outcome)
	}
	if d.missed > 1001 { // loop breaks the moment it exceeds 1000
		t.Fatalf("missed counter should be capped near 1000, got %d", d.missed)
	}
}

// newMockCron builds a Cron backed by a sqlmock database.
func newMockCron(t *testing.T) (*Cron, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	return &Cron{db: &db.DB{DB: raw}}, mock, raw
}

// TestClaim_WinsWhenRowMatches: when the conditional UPDATE matches the row
// (next_run_at unchanged), claim returns true — this worker owns the occurrence
// and would fire it.
func TestClaim_WinsWhenRowMatches(t *testing.T) {
	w, mock, raw := newMockCron(t)
	defer raw.Close()

	now := time.Now().UTC()
	prevNext := now.Add(-time.Minute)
	nextRun := now.Add(time.Minute)
	s := schedulable{kind: "function", id: "fn1", projectID: "p1", expr: "* * * * *"}

	mock.ExpectExec("UPDATE cron_state").
		WithArgs(now, nextRun, 0, s.kind, s.id, prevNext).
		WillReturnResult(sqlmock.NewResult(0, 1)) // one row updated

	if !w.claim(context.Background(), s, prevNext, nextRun, 0, now) {
		t.Fatal("claim should win when the row matches")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestClaim_LosesWhenAlreadyClaimed: a replay/second sweep for the same
// occurrence finds next_run_at already advanced, so the conditional UPDATE
// matches zero rows and claim returns false — the double-fire guard. Two sweeps
// against the same occurrence claim exactly once.
func TestClaim_LosesWhenAlreadyClaimed(t *testing.T) {
	w, mock, raw := newMockCron(t)
	defer raw.Close()

	now := time.Now().UTC()
	prevNext := now.Add(-time.Minute)
	nextRun := now.Add(time.Minute)
	s := schedulable{kind: "function", id: "fn1", projectID: "p1", expr: "* * * * *"}

	// First sweep wins the occurrence.
	mock.ExpectExec("UPDATE cron_state").
		WithArgs(now, nextRun, 0, s.kind, s.id, prevNext).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Second sweep (a replica, or a replay) claims the same occurrence: the row's
	// next_run_at no longer equals prevNext, so zero rows are affected.
	mock.ExpectExec("UPDATE cron_state").
		WithArgs(now, nextRun, 0, s.kind, s.id, prevNext).
		WillReturnResult(sqlmock.NewResult(0, 0))

	firstWon := w.claim(context.Background(), s, prevNext, nextRun, 0, now)
	secondWon := w.claim(context.Background(), s, prevNext, nextRun, 0, now)

	if !firstWon {
		t.Fatal("first claim should win")
	}
	if secondWon {
		t.Fatal("second claim must lose — the occurrence is already claimed (would double-fire)")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestClaim_CarriesMissedCount confirms the missed-run backlog is passed to the
// UPDATE so cron_state.missed_runs accumulates.
func TestClaim_CarriesMissedCount(t *testing.T) {
	w, mock, raw := newMockCron(t)
	defer raw.Close()

	now := time.Now().UTC()
	prevNext := now.Add(-24 * time.Hour)
	nextRun := now.Add(time.Hour)
	s := schedulable{kind: "deploy_target", id: "dt1", projectID: "p1", expr: "0 * * * *"}

	mock.ExpectExec("UPDATE cron_state").
		WithArgs(now, nextRun, 7, s.kind, s.id, prevNext).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if !w.claim(context.Background(), s, prevNext, nextRun, 7, now) {
		t.Fatal("claim should win")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
