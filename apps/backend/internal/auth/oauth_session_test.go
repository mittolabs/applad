package auth

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// An unverified OAuth email that collides with an existing account must be
// refused: no session, no link. This is the account-takeover guard.
func TestCreateOAuthSession_UnverifiedEmailCollisionRefused(t *testing.T) {
	svc, mock, db := newTestService(t)
	defer db.Close()

	// No identity match on (provider, oauth_id).
	mock.ExpectQuery("SELECT id FROM users WHERE project_id = .* AND oauth_provider").
		WillReturnError(sql.ErrNoRows)
	// An account already owns this email.
	mock.ExpectQuery("SELECT id FROM users WHERE project_id = .* AND email").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("existing-user"))

	_, _, err := svc.CreateOAuthSession(context.Background(), "proj", "github", "gh-123",
		"victim@example.com", "Attacker", false /* emailVerified */, "1.2.3.4", "ua")
	if err == nil {
		t.Fatal("expected refusal for unverified email collision")
	}
	if !strings.HasPrefix(err.Error(), "oauth_email_unverified") {
		t.Fatalf("expected oauth_email_unverified error, got: %v", err)
	}
	// Crucially, no UPDATE (link) and no session INSERT were performed.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected extra queries (a link or session must NOT happen): %v", err)
	}
}
