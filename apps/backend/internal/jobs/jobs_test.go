package jobs

import (
	"context"
	"fmt"
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

func TestCreateQueue_Defaults(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO job_queues").
		WillReturnResult(sqlmock.NewResult(1, 1))

	q, err := svc.CreateQueue(context.Background(), "proj1", "emails", "", 0, 0, 0, "")
	if err != nil {
		t.Fatalf("CreateQueue: %v", err)
	}
	if q.Concurrency != 10 {
		t.Errorf("concurrency default = %d, want 10", q.Concurrency)
	}
	if q.RetryLimit != 3 {
		t.Errorf("retryLimit default = %d, want 3", q.RetryLimit)
	}
	if q.RetryDelayS != 60 {
		t.Errorf("retryDelayS default = %d, want 60", q.RetryDelayS)
	}
}

func TestCreateQueue_DuplicateReturnsError(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO job_queues").
		WillReturnError(fmt.Errorf("Duplicate entry 'proj1-emails' for key 'uq_jq_project_name'"))

	_, err := svc.CreateQueue(context.Background(), "proj1", "emails", "", 10, 3, 60, "")
	if err == nil {
		t.Error("expected duplicate error")
	}
}

func TestEnqueue_SetsDefaults(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO jobs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	j, err := svc.Enqueue(context.Background(), "proj1", "q1", "send_email", nil, 0, time.Time{}, 0, nil)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if j.MaxAttempts != 3 {
		t.Errorf("maxAttempts = %d, want 3", j.MaxAttempts)
	}
	if j.Status != "pending" {
		t.Errorf("status = %s, want pending", j.Status)
	}
	if j.RunAt.IsZero() {
		t.Error("RunAt should not be zero")
	}
}

func TestEnqueue_RunAtFromParam(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	future := time.Now().Add(time.Hour)
	mock.ExpectExec("INSERT INTO jobs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	j, err := svc.Enqueue(context.Background(), "proj1", "q1", "scheduled_job", nil, 5, future, 1, nil)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if j.Priority != 5 {
		t.Errorf("priority = %d, want 5", j.Priority)
	}
}

func TestDeleteQueue(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("DELETE FROM job_queues").
		WithArgs("q1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.DeleteQueue(context.Background(), "q1", "proj1"); err != nil {
		t.Fatalf("DeleteQueue: %v", err)
	}
}

func TestCancelJob(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("UPDATE jobs SET status='cancelled'").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.CancelJob(context.Background(), "j1", "proj1"); err != nil {
		t.Fatalf("CancelJob: %v", err)
	}
}

func TestAck(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("UPDATE jobs SET status='completed'").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.Ack(context.Background(), "j1"); err != nil {
		t.Fatalf("Ack: %v", err)
	}
}

func TestRetry_RequeuesWithDelay(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("UPDATE jobs SET status='pending'").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.Retry(context.Background(), "j1", "boom", 30*time.Second); err != nil {
		t.Fatalf("Retry: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestFail_NoDeadLetter(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("UPDATE jobs SET status='failed'").
		WillReturnResult(sqlmock.NewResult(0, 1))

	job := &Job{ID: "j1", ProjectID: "proj1", Name: "send"}
	if err := svc.Fail(context.Background(), job, "boom", ""); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestFail_CopiesToDeadLetterQueue(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	// First the job is marked failed, then a copy is enqueued into the DLQ.
	mock.ExpectExec("UPDATE jobs SET status='failed'").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO jobs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	job := &Job{ID: "j1", ProjectID: "proj1", Name: "send", MaxAttempts: 3}
	if err := svc.Fail(context.Background(), job, "boom", "dlq1"); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestPushQueues(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "project_id", "name", "worker_url", "concurrency", "retry_limit",
		"retry_delay_s", "dead_letter_queue_id", "paused", "created_at", "updated_at",
	}).AddRow("q1", "proj1", "emails", "https://hook.example.com", 5, 3, 60, "", 0, now, now)
	mock.ExpectQuery("SELECT id, project_id, name, COALESCE\\(worker_url").
		WillReturnRows(rows)

	queues, err := svc.PushQueues(context.Background())
	if err != nil {
		t.Fatalf("PushQueues: %v", err)
	}
	if len(queues) != 1 || queues[0].WorkerURL != "https://hook.example.com" {
		t.Fatalf("PushQueues returned %+v", queues)
	}
}

func TestBoolIntHelper(t *testing.T) {
	if boolInt(true) != 1 || boolInt(false) != 0 {
		t.Error("boolInt mismatch")
	}
}
