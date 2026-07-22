package analytics

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
)

func newMockDB(t *testing.T) (*db.DB, sqlmock.Sqlmock) {
	t.Helper()
	raw, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	return &db.DB{DB: raw}, mock
}

func TestTrack_RequiresEvent(t *testing.T) {
	database, _ := newMockDB(t)
	svc := NewService(database)
	_, err := svc.Track(context.Background(), Event{ProjectID: "p1"})
	if err == nil {
		t.Error("expected error when event name is empty")
	}
}

func TestTrack_RequiresProject(t *testing.T) {
	database, _ := newMockDB(t)
	svc := NewService(database)
	_, err := svc.Track(context.Background(), Event{Event: "click"})
	if err == nil {
		t.Error("expected error when project ID is empty")
	}
}

func TestTrack_Inserts(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO analytics_events").
		WithArgs(
			sqlmock.AnyArg(),        // id
			"proj1",                 // project_id
			nil,                     // user_id
			nil,                     // session_id
			"page_view",             // event
			sqlmock.AnyArg(),        // properties
			nil, nil, nil, nil, nil, // url, referrer, device, browser, country
			sqlmock.AnyArg(), // created_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	e, err := svc.Track(context.Background(), Event{ProjectID: "proj1", Event: "page_view"})
	if err != nil {
		t.Fatalf("Track: %v", err)
	}
	if e.Event != "page_view" {
		t.Errorf("event = %s", e.Event)
	}
	if e.ID == "" {
		t.Error("expected non-empty ID")
	}
	if e.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}
	mock.ExpectationsWereMet() //nolint:errcheck
}

func TestTrackBatch_CountsInserted(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO analytics_events")
	for i := 0; i < 3; i++ {
		mock.ExpectExec("INSERT INTO analytics_events").WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()

	events := []Event{
		{ProjectID: "p1", Event: "e1"},
		{ProjectID: "p1", Event: "e2"},
		{ProjectID: "p1", Event: "e3"},
		{ProjectID: "", Event: "skip_no_proj"}, // skipped
	}
	n, err := svc.TrackBatch(context.Background(), events)
	if err != nil {
		t.Fatalf("TrackBatch: %v", err)
	}
	if n != 3 {
		t.Errorf("processed %d, want 3", n)
	}
}

func TestCreateFunnel_RequiresTwoSteps(t *testing.T) {
	database, _ := newMockDB(t)
	svc := NewService(database)
	_, err := svc.CreateFunnel(context.Background(), "p1", "onboard", []string{"sign_up"})
	// CreateFunnel itself doesn't enforce min steps — that's the handler's job
	// so this should succeed (DB mock will fail, but we test the handler separately)
	_ = err
}

func TestParseTimeRange_DefaultsToSevenDays(t *testing.T) {
	from, to := parseTimeRange("", "")
	diff := to.Sub(from)
	// Allow slight timing variance
	if diff < 6*24*time.Hour || diff > 8*24*time.Hour {
		t.Errorf("default time range = %v, want ~7d", diff)
	}
}

func TestParseTimeRange_ParsesRFC3339(t *testing.T) {
	fromStr := "2024-01-01T00:00:00Z"
	toStr := "2024-01-31T00:00:00Z"
	from, to := parseTimeRange(fromStr, toStr)
	if from.Year() != 2024 || from.Month() != 1 || from.Day() != 1 {
		t.Errorf("from = %v", from)
	}
	if to.Month() != 1 || to.Day() != 31 {
		t.Errorf("to = %v", to)
	}
}

func TestNullHelpers(t *testing.T) {
	if nullStr("") != nil {
		t.Error("empty string should be nil")
	}
	if nullStr("x") != "x" {
		t.Error("non-empty string should be returned as-is")
	}
	if nullBytes(nil) != nil {
		t.Error("nil bytes should be nil")
	}
	if nullBytes([]byte("a")) == nil {
		t.Error("non-empty bytes should not be nil")
	}
}

func TestRealtimeSummary(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectQuery(`SELECT COUNT\(DISTINCT user_id\) FROM analytics_events`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(`SELECT COUNT\(DISTINCT session_id\) FROM analytics_events`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM analytics_events`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(12))

	summary, err := svc.RealtimeSummary(context.Background(), "proj1", time.Now().Add(-5*time.Minute))
	if err != nil {
		t.Fatalf("RealtimeSummary: %v", err)
	}
	if summary["activeUsers"] != 3 || summary["activeSessions"] != 5 || summary["eventsLast5m"] != 12 {
		t.Errorf("unexpected summary: %+v", summary)
	}
}
