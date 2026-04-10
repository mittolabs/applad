package edge

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

var fnCols = []string{"id", "project_id", "name", "slug", "code", "runtime", "regions", "env_vars", "status", "created_at", "updated_at"}

func TestCreate_DefaultsRuntime(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO edge_functions").
		WillReturnResult(sqlmock.NewResult(1, 1))

	f, err := svc.Create(context.Background(), "proj1", "Auth Middleware", "auth-middleware",
		"export default (req) => req", "", nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if f.Runtime != "js" {
		t.Errorf("runtime = %s, want js", f.Runtime)
	}
	if f.Status != "draft" {
		t.Errorf("status = %s, want draft", f.Status)
	}
}

func TestCreate_Stores(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("INSERT INTO edge_functions").
		WillReturnResult(sqlmock.NewResult(1, 1))

	f, err := svc.Create(context.Background(), "proj1", "Geo Router", "geo-router",
		"export default (req) => { return fetch(req) }", "ts",
		[]string{"us-east-1", "eu-west-1"},
		map[string]string{"API_KEY": "secret"},
	)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if f.Slug != "geo-router" {
		t.Errorf("slug = %s", f.Slug)
	}
	if f.Runtime != "ts" {
		t.Errorf("runtime = %s", f.Runtime)
	}
}

func TestDelete(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("DELETE FROM edge_functions").
		WithArgs("fn1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.Delete(context.Background(), "fn1", "proj1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestDeploy_IncrementsVersion(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	now := time.Now()
	// Get function
	mock.ExpectQuery("SELECT .* FROM edge_functions WHERE id").
		WillReturnRows(sqlmock.NewRows(fnCols).
			AddRow("fn1", "proj1", "f", "f", "code", "js", `["us-east-1"]`, "{}", "deployed", now, now))
	// Max version lookup
	mock.ExpectQuery("SELECT COALESCE").
		WillReturnRows(sqlmock.NewRows([]string{"v"}).AddRow(2))
	// Insert deployment
	mock.ExpectExec("INSERT INTO edge_deployments").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Mark function deployed
	mock.ExpectExec("UPDATE edge_functions SET status='deployed'").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Mark deployment active
	mock.ExpectExec("UPDATE edge_deployments SET status='active'").
		WillReturnResult(sqlmock.NewResult(0, 1))

	d, err := svc.Deploy(context.Background(), "fn1", "proj1", nil)
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if d.Version != 3 {
		t.Errorf("version = %d, want 3 (prev was 2)", d.Version)
	}
	if d.Status != "active" {
		t.Errorf("status = %s, want active", d.Status)
	}
}

func TestList(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	now := time.Now()
	mock.ExpectQuery("SELECT .* FROM edge_functions WHERE project_id").
		WillReturnRows(sqlmock.NewRows(fnCols).
			AddRow("fn1", "proj1", "fn1", "fn-1", "", "js", "[]", "{}", "draft", now, now).
			AddRow("fn2", "proj1", "fn2", "fn-2", "", "ts", "[]", "{}", "deployed", now, now))

	fns, err := svc.List(context.Background(), "proj1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(fns) != 2 {
		t.Errorf("count = %d, want 2", len(fns))
	}
}

func TestEdgeSlug(t *testing.T) {
	if got := edgeSlug("/hello/world", "Hello World"); got != "hello-world" {
		t.Errorf("edgeSlug(route) = %s", got)
	}
	if got := edgeSlug("", "Hello World"); got != "hello-world" {
		t.Errorf("edgeSlug(name) = %s", got)
	}
}
