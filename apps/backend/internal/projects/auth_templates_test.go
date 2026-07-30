package projects

import (
	"context"
	"database/sql/driver"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
)

func newTemplateTestService(t *testing.T) (*Service, sqlmock.Sqlmock, func()) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	svc := NewService(&db.DB{DB: mockDB}, "", "test-secret")
	return svc, mock, func() { mockDB.Close() }
}

// capture records the driver value it matches, so the test can replay exactly
// the JSON the writer persisted into the subsequent read.
type capture struct{ got string }

func (c *capture) Match(v driver.Value) bool {
	switch s := v.(type) {
	case string:
		c.got = s
	case []byte:
		c.got = string(s)
	default:
		return false
	}
	return true
}

// Save a custom template, then read back exactly what was persisted — the
// roundtrip the auth mailer relies on. Unknown keys and empty templates must be
// dropped by the writer, not surface on read.
func TestAuthEmailTemplates_SaveThenGet(t *testing.T) {
	svc, mock, done := newTemplateTestService(t)
	defer done()

	cfg := &capture{}
	mock.ExpectQuery("SELECT auth_config FROM projects").
		WillReturnRows(sqlmock.NewRows([]string{"auth_config"}).AddRow(nil))
	mock.ExpectExec("UPDATE projects SET auth_config").
		WithArgs(cfg, "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	in := map[string]AuthEmailTemplate{
		"magic":       {Subject: "Sign in", Body: "<p>{{url}}</p>"},
		"unknown-key": {Subject: "nope", Body: "should be dropped"},
		"recovery":    {Subject: "", Body: ""}, // empty → dropped
	}
	if err := svc.UpdateAuthEmailTemplates(context.Background(), "proj1", in); err != nil {
		t.Fatalf("update: %v", err)
	}
	if cfg.got == "" {
		t.Fatal("writer persisted no auth_config")
	}

	// Get: replay the persisted JSON. Only the valid "magic" key survives.
	mock.ExpectQuery("SELECT auth_config FROM projects").
		WillReturnRows(sqlmock.NewRows([]string{"auth_config"}).AddRow(cfg.got))
	got, err := svc.GetAuthEmailTemplates(context.Background(), "proj1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly the magic template, got %d: %+v", len(got), got)
	}
	if got["magic"].Body != "<p>{{url}}</p>" || got["magic"].Subject != "Sign in" {
		t.Fatalf("magic template not persisted faithfully: %+v", got["magic"])
	}
	if _, ok := got["unknown-key"]; ok {
		t.Fatal("unknown key should have been dropped on write")
	}
	if _, ok := got["recovery"]; ok {
		t.Fatal("empty template should have been dropped on write")
	}

	// Resolve: the auth handler's view — ok for a saved key, not for a missing one.
	mock.ExpectQuery("SELECT auth_config FROM projects").
		WillReturnRows(sqlmock.NewRows([]string{"auth_config"}).AddRow(cfg.got))
	subject, body, ok := svc.AuthEmailTemplate(context.Background(), "proj1", "magic")
	if !ok || subject != "Sign in" || body != "<p>{{url}}</p>" {
		t.Fatalf("resolve magic: ok=%v subject=%q body=%q", ok, subject, body)
	}

	mock.ExpectQuery("SELECT auth_config FROM projects").
		WillReturnRows(sqlmock.NewRows([]string{"auth_config"}).AddRow(cfg.got))
	if _, _, ok := svc.AuthEmailTemplate(context.Background(), "proj1", "recovery"); ok {
		t.Fatal("recovery has no saved template; resolve must report ok=false")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
