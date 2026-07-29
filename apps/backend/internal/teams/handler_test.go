package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/middleware"
)

// projectCtxKeyType matches middleware.contextKey
type projectCtxKeyType int

func withProject(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), projectCtxKeyType(4), "test-project"))
		next.ServeHTTP(w, r)
	})
}

func TestCreateTeam_MissingName(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"empty body", `{}`, http.StatusBadRequest},
		{"empty name", `{"name":""}`, http.StatusBadRequest},
		{"whitespace name", `{"name":"  "}`, http.StatusBadRequest},
		{"invalid json", `bad`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux := chi.NewMux()
			mux.Use(withProject)
			mux.Post("/", h.createTeam)
			mux.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestCreateMembership_MissingEmail(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"empty body", `{}`, http.StatusBadRequest},
		{"no email", `{"roles":["admin"]}`, http.StatusBadRequest},
		{"empty email", `{"email":""}`, http.StatusBadRequest},
		{"invalid json", `bad`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/t1/memberships", bytes.NewReader([]byte(tc.body)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux := chi.NewMux()
			mux.Use(withProject)
			mux.Post("/{teamId}/memberships", h.createMembership)
			mux.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Fatalf("expected %d, got %d: %s", tc.want, w.Code, w.Body.String())
			}
		})
	}
}

func TestRoutes_Structure(t *testing.T) {
	svc := &Service{}
	h := NewHandler(svc)
	router := Routes(h)
	if router == nil {
		t.Fatal("expected non-nil router")
	}
}

// --- Authorization tests (P0: broken access control on team operations) ------

const membershipsSelect = `SELECT roles FROM memberships WHERE team_id = \$1 AND user_id = \$2 AND joined = TRUE`

func newMockHandler(t *testing.T) (*Handler, sqlmock.Sqlmock, func()) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	h := NewHandler(NewService(&db.DB{DB: mockDB}))
	return h, mock, func() { mockDB.Close() }
}

func serve(mount func(*chi.Mux), method, target, body string, ctx context.Context) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := chi.NewMux()
	mount(mux)
	mux.ServeHTTP(w, req)
	return w
}

func userCtx(userID string) context.Context {
	return middleware.ContextWithProject(middleware.ContextWithUser(context.Background(), userID), "proj1")
}

func apiKeyCtx() context.Context {
	return middleware.ContextWithProject(middleware.ContextWithAPIKey(context.Background()), "proj1")
}

// A non-member must not be able to read a team's roster (which would leak every
// member and their email): they get a 403 and the roster query is never run.
func TestListMemberships_NonMemberForbidden(t *testing.T) {
	h, mock, done := newMockHandler(t)
	defer done()

	mock.ExpectQuery(membershipsSelect).
		WithArgs("t1", "attacker").
		WillReturnRows(sqlmock.NewRows([]string{"roles"})) // no membership row

	w := serve(func(m *chi.Mux) { m.Get("/{teamId}/memberships", h.listMemberships) },
		http.MethodGet, "/t1/memberships", "", userCtx("attacker"))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations (roster must not be queried): %v", err)
	}
}

// A joined member may list the roster.
func TestListMemberships_MemberAllowed(t *testing.T) {
	h, mock, done := newMockHandler(t)
	defer done()

	mock.ExpectQuery(membershipsSelect).
		WithArgs("t1", "member1").
		WillReturnRows(sqlmock.NewRows([]string{"roles"}).AddRow([]byte(`["member"]`)))
	mock.ExpectQuery(`SELECT m.id, m.team_id, m.user_id`).
		WithArgs("t1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "team_id", "user_id", "invited_email", "roles", "invited", "joined", "created_at", "name"}).
			AddRow("m1", "t1", "member1", "a@b.c", []byte(`["member"]`), true, true, timeNow(), "Team"))

	w := serve(func(m *chi.Mux) { m.Get("/{teamId}/memberships", h.listMemberships) },
		http.MethodGet, "/t1/memberships", "", userCtx("member1"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// A member who is not an owner cannot rename the team.
func TestUpdateTeam_NonOwnerForbidden(t *testing.T) {
	h, mock, done := newMockHandler(t)
	defer done()

	mock.ExpectQuery(membershipsSelect).
		WithArgs("t1", "member1").
		WillReturnRows(sqlmock.NewRows([]string{"roles"}).AddRow([]byte(`["member"]`)))

	w := serve(func(m *chi.Mux) { m.Put("/{teamId}", h.updateTeam) },
		http.MethodPut, "/t1", `{"name":"Hijacked"}`, userCtx("member1"))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations (update must not run): %v", err)
	}
}

// An owner can rename the team.
func TestUpdateTeam_OwnerAllowed(t *testing.T) {
	h, mock, done := newMockHandler(t)
	defer done()

	mock.ExpectQuery(membershipsSelect).
		WithArgs("t1", "owner1").
		WillReturnRows(sqlmock.NewRows([]string{"roles"}).AddRow([]byte(`["owner"]`)))
	mock.ExpectExec(`UPDATE teams SET name = \$1 WHERE id = \$2 AND project_id = \$3`).
		WithArgs("Renamed", "t1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectGet(mock, "t1", "proj1")

	w := serve(func(m *chi.Mux) { m.Put("/{teamId}", h.updateTeam) },
		http.MethodPut, "/t1", `{"name":"Renamed"}`, userCtx("owner1"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// A server API key keeps full access: no membership check, the update runs.
func TestUpdateTeam_APIKeyAllowed(t *testing.T) {
	h, mock, done := newMockHandler(t)
	defer done()

	mock.ExpectExec(`UPDATE teams SET name = \$1 WHERE id = \$2 AND project_id = \$3`).
		WithArgs("Renamed", "t1", "proj1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectGet(mock, "t1", "proj1")

	w := serve(func(m *chi.Mux) { m.Put("/{teamId}", h.updateTeam) },
		http.MethodPut, "/t1", `{"name":"Renamed"}`, apiKeyCtx())

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// The escalation path: a non-member invites themselves (with any roles) so they
// can accept the returned secret and land inside a privileged team. Must 403,
// and no membership must be created.
func TestCreateMembership_SelfAddNonMemberBlocked(t *testing.T) {
	h, mock, done := newMockHandler(t)
	defer done()

	mock.ExpectQuery(membershipsSelect).
		WithArgs("t1", "attacker").
		WillReturnRows(sqlmock.NewRows([]string{"roles"})) // not a member

	w := serve(func(m *chi.Mux) { m.Post("/{teamId}/memberships", h.createMembership) },
		http.MethodPost, "/t1/memberships", `{"email":"attacker@x.com","roles":["owner"]}`, userCtx("attacker"))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations (no membership must be inserted): %v", err)
	}
}

// A non-member cannot delete another team.
func TestDeleteTeam_NonMemberForbidden(t *testing.T) {
	h, mock, done := newMockHandler(t)
	defer done()

	mock.ExpectQuery(membershipsSelect).
		WithArgs("t1", "attacker").
		WillReturnRows(sqlmock.NewRows([]string{"roles"}))

	w := serve(func(m *chi.Mux) { m.Delete("/{teamId}", h.deleteTeam) },
		http.MethodDelete, "/t1", "", userCtx("attacker"))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations (delete must not run): %v", err)
	}
}

func timeNow() time.Time { return time.Now().UTC().Truncate(time.Second) }

func expectGet(mock sqlmock.Sqlmock, teamID, projectID string) {
	mock.ExpectQuery(`SELECT id, name, prefs, created_at, updated_at FROM teams WHERE`).
		WithArgs(teamID, projectID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "prefs", "created_at", "updated_at"}).
			AddRow(teamID, "Renamed", []byte(`{}`), timeNow(), timeNow()))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM memberships WHERE team_id = \$1`).
		WithArgs(teamID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]int{"total": 0})

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result map[string]int
	json.NewDecoder(w.Body).Decode(&result)
	if result["total"] != 0 {
		t.Fatalf("expected total=0, got %d", result["total"])
	}
}
