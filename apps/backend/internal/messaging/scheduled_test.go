package messaging

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
)

func newMockService(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	return NewService(&db.DB{DB: mockDB}, Config{}), mock
}

// sendDecision is what the handler consults before touching the DB: a future
// scheduledAt must queue (no inline send), an absent or past one must send now.
func TestSendDecision(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	cases := []struct {
		name        string
		draft       bool
		scheduledAt *time.Time
		wantStatus  string
		wantSendNow bool
	}{
		{"immediate", false, nil, "processing", true},
		{"future is queued, not sent", false, &future, "scheduled", false},
		{"past sends immediately", false, &past, "processing", true},
		{"draft never sends", true, &future, "draft", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, sendNow := sendDecision(tc.draft, tc.scheduledAt)
			if status != tc.wantStatus || sendNow != tc.wantSendNow {
				t.Fatalf("got (%q,%v), want (%q,%v)", status, sendNow, tc.wantStatus, tc.wantSendNow)
			}
		})
	}
}

// A future scheduledAt is persisted with status 'scheduled' and its scheduled_at
// column set — proving the record is stored without an inline send.
func TestCreateMessage_Scheduled_PersistsScheduledAt(t *testing.T) {
	svc, mock := newMockService(t)
	defer mock.ExpectationsWereMet() //nolint:errcheck

	when := time.Now().Add(2 * time.Hour).UTC()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO messages`)).
		WithArgs(sqlmock.AnyArg(), "proj1", "email", "Hi", "<p>", `["a@b.com"]`, "scheduled", when).
		WillReturnResult(sqlmock.NewResult(1, 1))

	msg, err := svc.CreateMessage(context.Background(), "proj1", "email", "Hi", "<p>", []string{"a@b.com"}, "scheduled", &when)
	if err != nil {
		t.Fatalf("CreateMessage: %v", err)
	}
	if msg.Status != "scheduled" {
		t.Fatalf("status = %q, want scheduled", msg.Status)
	}
	if msg.ScheduledAt == nil {
		t.Fatal("ScheduledAt not set on returned message")
	}
}

// A due message is selected, claimed via a status transition, delivered, and
// marked sent. The claim's conditional WHERE (status='scheduled') is what makes
// it exactly-once.
func TestSweepScheduledMessages_DueMessageMarkedSent(t *testing.T) {
	svc, mock := newMockService(t)
	defer mock.ExpectationsWereMet() //nolint:errcheck

	rows := sqlmock.NewRows([]string{"id", "project_id", "type", "subject", "body", "recipients"}).
		AddRow("m1", "proj1", "email", "Hi", "<p>", `["a@b.com"]`)
	mock.ExpectQuery(regexp.QuoteMeta("WHERE status = 'scheduled' AND scheduled_at IS NOT NULL AND scheduled_at <= NOW()")).
		WillReturnRows(rows)
	// Claim: scheduled -> processing (one row affected == we own the send).
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE messages SET status = 'processing' WHERE id = $1 AND status = 'scheduled'`)).
		WithArgs("m1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Delivered successfully -> status becomes sent, delivered_at stamped.
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE messages SET status=$1, delivered_at=NOW() WHERE id=$2`)).
		WithArgs("sent", "m1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	var delivered *Message
	n, err := svc.SweepScheduledMessages(context.Background(), func(_ context.Context, m *Message) error {
		delivered = m
		return nil
	})
	if err != nil {
		t.Fatalf("SweepScheduledMessages: %v", err)
	}
	if n != 1 {
		t.Fatalf("claimed = %d, want 1", n)
	}
	if delivered == nil || delivered.ID != "m1" {
		t.Fatalf("expected message m1 delivered, got %+v", delivered)
	}
}

// A due message already claimed by another sweeper (the conditional UPDATE
// matches no row) is skipped: not delivered, not re-statused.
func TestSweepScheduledMessages_AlreadyClaimed_Skipped(t *testing.T) {
	svc, mock := newMockService(t)
	defer mock.ExpectationsWereMet() //nolint:errcheck

	rows := sqlmock.NewRows([]string{"id", "project_id", "type", "subject", "body", "recipients"}).
		AddRow("m1", "proj1", "email", "Hi", "<p>", `["a@b.com"]`)
	mock.ExpectQuery(regexp.QuoteMeta("WHERE status = 'scheduled'")).WillReturnRows(rows)
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE messages SET status = 'processing' WHERE id = $1 AND status = 'scheduled'`)).
		WithArgs("m1").
		WillReturnResult(sqlmock.NewResult(0, 0)) // lost the race

	called := false
	n, err := svc.SweepScheduledMessages(context.Background(), func(_ context.Context, _ *Message) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("SweepScheduledMessages: %v", err)
	}
	if n != 0 {
		t.Fatalf("claimed = %d, want 0", n)
	}
	if called {
		t.Fatal("deliver should not be called for a message we did not claim")
	}
}
