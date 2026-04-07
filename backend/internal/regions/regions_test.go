package regions

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

var regionCols = []string{"id", "name", "code", "location", "endpoint", "latitude", "longitude", "status", "created_at"}

func TestListRegions(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	now := time.Now()
	mock.ExpectQuery("SELECT .* FROM regions WHERE status = 'active'").
		WillReturnRows(sqlmock.NewRows(regionCols).
			AddRow("r1", "US East", "us-east-1", "Virginia", "", 38.9, -77.0, "active", now).
			AddRow("r2", "EU West", "eu-west-1", "Dublin", "", 53.3, -6.2, "active", now))

	regions, err := svc.ListRegions(context.Background())
	if err != nil {
		t.Fatalf("ListRegions: %v", err)
	}
	if len(regions) != 2 {
		t.Errorf("count = %d, want 2", len(regions))
	}
	if regions[0].Code != "us-east-1" {
		t.Errorf("code = %s", regions[0].Code)
	}
}

func TestGetRegion_ByCode(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	now := time.Now()
	mock.ExpectQuery("SELECT .* FROM regions WHERE id = .* OR code = .*").
		WithArgs("us-east-1", "us-east-1").
		WillReturnRows(sqlmock.NewRows(regionCols).
			AddRow("r1", "US East", "us-east-1", "Virginia", "", 38.9, -77.0, "active", now))

	r, err := svc.GetRegion(context.Background(), "us-east-1")
	if err != nil {
		t.Fatalf("GetRegion: %v", err)
	}
	if r.Name != "US East" {
		t.Errorf("name = %s", r.Name)
	}
}

func TestAssignRegion_ClearsPrimary(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	// Expects primary clear
	mock.ExpectExec("UPDATE project_regions SET primary_region=0").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Expects upsert
	mock.ExpectExec("INSERT INTO project_regions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Expects region lookup
	now := time.Now()
	mock.ExpectQuery("SELECT .* FROM regions WHERE id = .* OR code = .*").
		WillReturnRows(sqlmock.NewRows(regionCols).
			AddRow("r1", "US East", "us-east-1", "Virginia", "", 38.9, -77.0, "active", now))

	pr, err := svc.AssignRegion(context.Background(), "proj1", "r1", true, true, false)
	if err != nil {
		t.Fatalf("AssignRegion: %v", err)
	}
	if !pr.PrimaryRegion {
		t.Error("primary should be true")
	}
	if !pr.GDPR {
		t.Error("gdpr should be true")
	}
	if pr.HIPAA {
		t.Error("hipaa should be false")
	}
}

func TestAssignRegion_NoPrimaryClear(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	// No primary clear expected
	mock.ExpectExec("INSERT INTO project_regions").
		WillReturnResult(sqlmock.NewResult(1, 1))
	now := time.Now()
	mock.ExpectQuery("SELECT .* FROM regions WHERE id = .* OR code = .*").
		WillReturnRows(sqlmock.NewRows(regionCols).
			AddRow("r2", "EU West", "eu-west-1", "Dublin", "", 53.3, -6.2, "active", now))

	_, err := svc.AssignRegion(context.Background(), "proj1", "r2", false, false, false)
	if err != nil {
		t.Fatalf("AssignRegion: %v", err)
	}
}

func TestRemoveRegion(t *testing.T) {
	database, mock := newMockDB(t)
	svc := NewService(database)

	mock.ExpectExec("DELETE FROM project_regions").
		WithArgs("proj1", "r1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.RemoveRegion(context.Background(), "proj1", "r1"); err != nil {
		t.Fatalf("RemoveRegion: %v", err)
	}
}

func TestBoolInt(t *testing.T) {
	if boolInt(true) != 1 || boolInt(false) != 0 {
		t.Error("boolInt mismatch")
	}
}
