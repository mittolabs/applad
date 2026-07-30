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

// TestCreate_RolesOnOwnerMembership proves create(..., roles) is no longer inert:
// the creator's owner membership records the union of {"owner"} and the supplied
// roles, with "owner" first and deduplicated.
func TestCreate_RolesOnOwnerMembership(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	mock.ExpectExec(`INSERT INTO teams`).
		WithArgs(sqlmock.AnyArg(), "proj1", "Engineering", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// The roles column is the union: owner is guaranteed and kept first, the
	// caller's "developer" rides along, and a redundant "owner" is dropped.
	mock.ExpectExec(`INSERT INTO memberships`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "u1", []byte(`["owner","developer"]`), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	_, err := svc.Create(context.Background(), "proj1", "", "Engineering", "u1", []string{"owner", "developer"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestUpdateMembershipRoles_RoundTrip proves the roles column is written whole
// and the refreshed membership is read back.
func TestUpdateMembershipRoles_RoundTrip(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	mock.ExpectExec(`UPDATE memberships SET roles = \$1 WHERE id = \$2 AND team_id = \$3`).
		WithArgs([]byte(`["developer","lead"]`), "mem1", "t1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT m.id, m.team_id, m.user_id, m.invited_email, m.roles, m.invited, m.joined, m.created_at, t.name FROM memberships m JOIN teams t ON t.id = m.team_id WHERE m.id = \$1 AND m.team_id = \$2 AND t.project_id = \$3`).
		WithArgs("mem1", "t1", "proj1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "team_id", "user_id", "invited_email", "roles", "invited", "joined", "created_at", "name"}).
			AddRow("mem1", "t1", "u2", "g@x.com", []byte(`["developer","lead"]`), true, true, time.Now().UTC(), "Team"))

	m, err := svc.UpdateMembershipRoles(context.Background(), "t1", "mem1", "proj1", []string{"developer", "lead"})
	if err != nil {
		t.Fatalf("UpdateMembershipRoles: %v", err)
	}
	if len(m.Roles) != 2 || m.Roles[0] != "developer" || m.Roles[1] != "lead" {
		t.Fatalf("expected [developer lead], got %v", m.Roles)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestUpdateMembershipRoles_NotFound maps a missing row to "membership not found".
func TestUpdateMembershipRoles_NotFound(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	mock.ExpectExec(`UPDATE memberships SET roles`).
		WithArgs(sqlmock.AnyArg(), "missing", "t1", "proj123").
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := svc.UpdateMembershipRoles(context.Background(), "t1", "missing", "proj123", []string{"x"})
	if err == nil || err.Error() != "membership not found" {
		t.Fatalf("expected membership not found, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestDeleteMembership_ProjectScoped proves DeleteMembership's projectID is
// load-bearing: the DELETE carries it into an EXISTS clause on teams, so a call
// scoped to the wrong project matches no row (and removes nothing) rather than
// deleting a membership in another project's team.
func TestDeleteMembership_ProjectScoped(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	mock.ExpectExec(`DELETE FROM memberships WHERE id = \$1 AND team_id = \$2 AND EXISTS \(SELECT 1 FROM teams WHERE teams.id = \$2 AND teams.project_id = \$3\)`).
		WithArgs("mem1", "t1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 0)) // cross-project: nothing deleted

	if err := svc.DeleteMembership(context.Background(), "mem1", "t1", "proj1"); err != nil {
		t.Fatalf("DeleteMembership: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestUpdateMembershipRoles_CrossProjectNotFound proves a role update aimed at a
// team in another project affects no row (the EXISTS clause fails) and surfaces
// as "membership not found" rather than silently succeeding.
func TestUpdateMembershipRoles_CrossProjectNotFound(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	mock.ExpectExec(`UPDATE memberships SET roles = \$1 WHERE id = \$2 AND team_id = \$3 AND EXISTS`).
		WithArgs([]byte(`["owner"]`), "mem1", "t1", "wrong-project").
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := svc.UpdateMembershipRoles(context.Background(), "t1", "mem1", "wrong-project", []string{"owner"})
	if err == nil || err.Error() != "membership not found" {
		t.Fatalf("expected membership not found, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
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
