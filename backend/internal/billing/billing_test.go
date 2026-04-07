package billing

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

func TestCreatePlan_Inserts(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO billing_plans").
		WillReturnResult(sqlmock.NewResult(1, 1))

	p, err := svc.CreatePlan(context.Background(), "Pro", "pro", 2900, 29000,
		map[string]interface{}{"api_calls": 100000},
		[]string{"databases", "storage", "functions"},
	)
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	if p.Slug != "pro" {
		t.Errorf("slug = %s", p.Slug)
	}
	if p.PriceMonthly != 2900 {
		t.Errorf("price = %d, want 2900", p.PriceMonthly)
	}
	if !p.Active {
		t.Error("new plan should be active")
	}
}

func TestSubscribe_InsertsOrUpdates(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO billing_subscriptions").
		WillReturnResult(sqlmock.NewResult(1, 1))

	sub, err := svc.Subscribe(context.Background(), "proj1", "plan1")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if sub.Status != "active" {
		t.Errorf("status = %s, want active", sub.Status)
	}
	if sub.CurrentPeriodStart == nil {
		t.Error("CurrentPeriodStart should not be nil")
	}
	if sub.CurrentPeriodEnd == nil {
		t.Error("CurrentPeriodEnd should not be nil")
	}
	// Period should be approximately 1 month
	diff := sub.CurrentPeriodEnd.Sub(*sub.CurrentPeriodStart)
	if diff < 28*24*time.Hour {
		t.Errorf("billing period too short: %v", diff)
	}
}

func TestCancelSubscription(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("UPDATE billing_subscriptions SET cancel_at_period_end=1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.CancelSubscription(context.Background(), "proj1"); err != nil {
		t.Fatalf("CancelSubscription: %v", err)
	}
}

func TestRecordEvent(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO billing_events").
		WillReturnResult(sqlmock.NewResult(1, 1))

	e, err := svc.RecordEvent(context.Background(), "proj1", "api_call", 1, "request", nil)
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	if e.EventType != "api_call" {
		t.Errorf("eventType = %s", e.EventType)
	}
	if e.Quantity != 1 {
		t.Errorf("quantity = %d, want 1", e.Quantity)
	}
}

func TestGetUsageSummary_Empty(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectQuery("SELECT event_type, SUM").
		WillReturnRows(sqlmock.NewRows([]string{"event_type", "qty"}))

	now := time.Now()
	summary, err := svc.GetUsageSummary(context.Background(), "proj1", now.AddDate(0, -1, 0), now)
	if err != nil {
		t.Fatalf("GetUsageSummary: %v", err)
	}
	if summary.Total != 0 {
		t.Errorf("total = %d, want 0", summary.Total)
	}
	if len(summary.Events) != 0 {
		t.Errorf("events should be empty map")
	}
}

func TestGetUsageSummary_Aggregates(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	rows := sqlmock.NewRows([]string{"event_type", "SUM(quantity)"}).
		AddRow("api_call", 5000).
		AddRow("storage_bytes", 1024)
	mock.ExpectQuery("SELECT event_type, SUM").WillReturnRows(rows)

	now := time.Now()
	summary, err := svc.GetUsageSummary(context.Background(), "proj1", now.AddDate(0, -1, 0), now)
	if err != nil {
		t.Fatalf("GetUsageSummary: %v", err)
	}
	if summary.Events["api_call"] != 5000 {
		t.Errorf("api_call = %d, want 5000", summary.Events["api_call"])
	}
	if summary.Total != 6024 {
		t.Errorf("total = %d, want 6024", summary.Total)
	}
}

func TestListPlans_Empty(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectQuery("SELECT id, name, slug").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "price_monthly", "price_yearly", "limits", "features", "active", "created_at", "updated_at"}))

	plans, err := svc.ListPlans(context.Background())
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if plans != nil {
		t.Errorf("expected nil slice for empty result, got %v", plans)
	}
}
