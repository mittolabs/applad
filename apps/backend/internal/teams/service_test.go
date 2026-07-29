package teams

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
)

func newTestService(t *testing.T) (*Service, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	return NewService(&db.DB{DB: mockDB}), mock, mockDB
}

// TestUpdatePrefs_RoundTrip proves a team's prefs are written and then read back
// unchanged: the UPDATE persists the JSON, and the following Get returns the same
// object rather than a hardcoded empty map.
func TestUpdatePrefs_RoundTrip(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	const teamID = "team123"
	const projectID = "proj123"
	now := time.Now().UTC().Truncate(time.Second)
	prefsStored := []byte(`{"color":"blue","notifications":true}`)

	// UPDATE writes the prefs JSON for this team/project.
	mock.ExpectExec(`UPDATE teams SET prefs = \$1, updated_at = \$2 WHERE id = \$3 AND project_id = \$4`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), teamID, projectID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Get re-reads the row (now carrying the stored prefs) plus the member count.
	mock.ExpectQuery(`SELECT id, name, prefs, created_at, updated_at FROM teams WHERE`).
		WithArgs(teamID, projectID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "prefs", "created_at", "updated_at"}).
			AddRow(teamID, "Engineering", prefsStored, now, now))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM memberships WHERE team_id = \$1`).
		WithArgs(teamID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	prefs := map[string]interface{}{"color": "blue", "notifications": true}
	got, err := svc.UpdatePrefs(context.Background(), teamID, projectID, prefs)
	if err != nil {
		t.Fatalf("UpdatePrefs: %v", err)
	}
	if got.Prefs["color"] != "blue" {
		t.Fatalf("expected color=blue, got %v", got.Prefs["color"])
	}
	if got.Prefs["notifications"] != true {
		t.Fatalf("expected notifications=true, got %v", got.Prefs["notifications"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestMembershipOf classifies a caller as non-member, plain member, or owner
// from their joined membership row — the check every team mutation relies on.
func TestMembershipOf(t *testing.T) {
	cases := []struct {
		name       string
		rows       *sqlmock.Rows
		wantMember bool
		wantOwner  bool
	}{
		{"non-member", sqlmock.NewRows([]string{"roles"}), false, false},
		{"plain member", sqlmock.NewRows([]string{"roles"}).AddRow([]byte(`["member"]`)), true, false},
		{"owner", sqlmock.NewRows([]string{"roles"}).AddRow([]byte(`["owner","member"]`)), true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, mock, mockDB := newTestService(t)
			defer mockDB.Close()
			mock.ExpectQuery(`SELECT roles FROM memberships WHERE team_id = \$1 AND user_id = \$2 AND joined = TRUE`).
				WithArgs("t1", "u1").
				WillReturnRows(tc.rows)
			member, owner, err := svc.MembershipOf(context.Background(), "t1", "u1")
			if err != nil {
				t.Fatalf("MembershipOf: %v", err)
			}
			if member != tc.wantMember || owner != tc.wantOwner {
				t.Fatalf("got member=%v owner=%v, want member=%v owner=%v", member, owner, tc.wantMember, tc.wantOwner)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("unmet expectations: %v", err)
			}
		})
	}
}

// TestUpdatePrefs_NotFound maps a missing team (no rows updated) to a "team not
// found" error, which the handler turns into a 404.
func TestUpdatePrefs_NotFound(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	mock.ExpectExec(`UPDATE teams SET prefs`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "missing", "proj123").
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := svc.UpdatePrefs(context.Background(), "missing", "proj123", map[string]interface{}{"x": 1})
	if err == nil || err.Error() != "team not found" {
		t.Fatalf("expected team not found, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
