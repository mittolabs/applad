package workflows

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/middleware"
)

func newMockHandler(t *testing.T) (*Handler, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	database := &db.DB{DB: mockDB}
	return NewHandler(NewService(database, nil)), mock
}

func workflowRow(id, projectID string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "project_id", "name", "description", "status", "trigger_type",
		"trigger_config", "webhook_secret", "nodes", "edges", "created_at",
		"updated_at", "error_workflow_id", "retry_attempts", "retry_delay_ms",
	}).AddRow(id, projectID, "W", "", "active", "manual",
		[]byte("{}"), "", []byte("[]"), []byte("[]"), time.Now(), time.Now(),
		"", int64(0), int64(0))
}

// --- P1: cross-project IDOR on versions/shares ---

// Project A must not read project B's workflow version history. The scope check
// runs a project-scoped Get first; a cross-project id returns no row, so the
// handler 404s and never reaches ListVersions.
func TestListVersions_CrossProjectDenied(t *testing.T) {
	h, mock := newMockHandler(t)
	// Project A asks for workflow "wfB" (owned by B). Project-scoped lookup misses.
	mock.ExpectQuery(`FROM workflows WHERE id = \$1 AND project_id = \$2`).
		WithArgs("wfB", "projA").WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/wfB/versions", nil)
	req.Header.Set("X-Applad-Project", "projA")
	rr := httptest.NewRecorder()
	middleware.ProjectContext(Routes(h)).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("cross-project read: expected 404, got %d (%s)", rr.Code, rr.Body.String())
	}
	// If ListVersions had run, sqlmock would have flagged its query as unexpected.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet/extra expectations: %v", err)
	}
}

// The legitimate owner still reads its own versions.
func TestListVersions_OwnerAllowed(t *testing.T) {
	h, mock := newMockHandler(t)
	mock.ExpectQuery(`FROM workflows WHERE id = \$1 AND project_id = \$2`).
		WithArgs("wfA", "projA").WillReturnRows(workflowRow("wfA", "projA"))
	mock.ExpectQuery(`FROM workflow_versions WHERE workflow_id=\$1`).
		WithArgs("wfA").WillReturnRows(
		sqlmock.NewRows([]string{"id", "version", "name", "created_at", "created_by"}).
			AddRow("v1", 1, "W", time.Now(), "u1"))

	req := httptest.NewRequest(http.MethodGet, "/wfA/versions", nil)
	req.Header.Set("X-Applad-Project", "projA")
	rr := httptest.NewRecorder()
	middleware.ProjectContext(Routes(h)).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("owner read: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "\"version\"") {
		t.Errorf("expected versions in body, got %s", rr.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// Listing shares is likewise project-scoped.
func TestListShares_CrossProjectDenied(t *testing.T) {
	h, mock := newMockHandler(t)
	mock.ExpectQuery(`FROM workflows WHERE id = \$1 AND project_id = \$2`).
		WithArgs("wfB", "projA").WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/wfB/shares", nil)
	req.Header.Set("X-Applad-Project", "projA")
	rr := httptest.NewRecorder()
	middleware.ProjectContext(Routes(h)).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("cross-project shares: expected 404, got %d", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet/extra expectations: %v", err)
	}
}

// --- P2: Update mints a webhook secret on transition into webhook trigger ---

func TestUpdate_TransitionToWebhookSetsSecret(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(&db.DB{DB: mockDB}, nil)

	// Main update.
	mock.ExpectExec(`UPDATE workflows SET name=`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Conditional secret update (only fires for webhook trigger, when absent).
	mock.ExpectExec(`SET webhook_secret=\$1 WHERE id=\$2 AND project_id=\$3 AND \(webhook_secret IS NULL OR webhook_secret=''\)`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Final Get.
	mock.ExpectQuery(`FROM workflows WHERE id = \$1 AND project_id = \$2`).
		WithArgs("wf1", "proj1").WillReturnRows(workflowRow("wf1", "proj1"))

	_, err = svc.Update(context.Background(), "wf1", "proj1", "W", "", "active", "webhook", nil, nil, nil)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expected the webhook-secret update to fire: %v", err)
	}
}

func TestUpdate_ManualTriggerNoSecretUpdate(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(&db.DB{DB: mockDB}, nil)

	mock.ExpectExec(`UPDATE workflows SET name=`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`FROM workflows WHERE id = \$1 AND project_id = \$2`).
		WithArgs("wf1", "proj1").WillReturnRows(workflowRow("wf1", "proj1"))

	// No secret-update expectation: a manual trigger must not issue one.
	if _, err = svc.Update(context.Background(), "wf1", "proj1", "W", "", "active", "manual", nil, nil, nil); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet/extra expectations: %v", err)
	}
}
