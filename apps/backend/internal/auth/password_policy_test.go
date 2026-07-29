package auth

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"golang.org/x/crypto/bcrypt"
)

// securityConfigRows builds the single-column auth_config result that
// loadSecurity reads, wrapping the given security JSON under the "security" key.
func securityConfigRows(securityJSON string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"auth_config"}).
		AddRow(`{"security":` + securityJSON + `}`)
}

// --- pure helpers ---

func TestPasswordContainsPersonalData(t *testing.T) {
	cases := []struct {
		name, password, email, uname string
		want                         bool
	}{
		{"email local part case-insensitive", "GraceHopper2024", "grace@example.com", "", true},
		{"email local part hit", "mygrace-pw99", "grace@example.com", "", true},
		{"name word hit", "hopper-is-cool", "x@y.com", "Grace Hopper", true},
		{"dotted local part", "the-hopper-1", "grace.hopper@example.com", "", true},
		{"clean password", "b7#Kq!vm2Zt", "grace@example.com", "Grace Hopper", false},
		{"short token ignored", "ada-power", "a@b.com", "Ada", true}, // "ada" >= 3 so this hits
		{"two-letter token ignored", "loves-io-99", "", "Io", false}, // "io" < 3 ignored
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := passwordContainsPersonalData(c.password, c.email, c.uname)
			if got != c.want {
				t.Errorf("passwordContainsPersonalData(%q,%q,%q) = %v, want %v",
					c.password, c.email, c.uname, got, c.want)
			}
		})
	}
}

func TestIsCommonPassword(t *testing.T) {
	for _, p := range []string{"password", "PASSWORD", "123456", "Qwerty", "letmein"} {
		if !isCommonPassword(p) {
			t.Errorf("expected %q to be flagged as common", p)
		}
	}
	for _, p := range []string{"b7#Kq!vm2Zt", "correct-horse-battery-staple"} {
		if isCommonPassword(p) {
			t.Errorf("expected %q to be allowed", p)
		}
	}
}

func TestShouldSendSessionAlert(t *testing.T) {
	cases := []struct {
		name      string
		enabled   bool
		provider  string
		email     string
		totalSess int
		want      bool
	}{
		{"disabled", false, "email", "a@b.com", 3, false},
		{"password later session", true, "email", "a@b.com", 3, true},
		{"first session skipped", true, "email", "a@b.com", 1, false},
		{"magic link skipped", true, "magic_link", "a@b.com", 3, false},
		{"oauth skipped", true, "google", "a@b.com", 3, false},
		{"phone skipped", true, "phone", "a@b.com", 3, false},
		{"anonymous email skipped", true, "email", "anon_x@anonymous.applad.local", 3, false},
		{"no email skipped", true, "email", "", 3, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := shouldSendSessionAlert(c.enabled, c.provider, c.email, c.totalSess)
			if got != c.want {
				t.Errorf("shouldSendSessionAlert(%v,%q,%q,%d) = %v, want %v",
					c.enabled, c.provider, c.email, c.totalSess, got, c.want)
			}
		})
	}
}

// --- CreateAccount: personal data rejected ---

func TestCreateAccount_PersonalDataRejected(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	mock.ExpectQuery(`SELECT auth_config FROM projects WHERE`).
		WithArgs(testProjectID).
		WillReturnRows(securityConfigRows(`{"passwordPersonalData":true}`))

	// Password embeds the user's name; no INSERT should be attempted.
	_, err := svc.CreateAccount(context.Background(), testProjectID, "unique()",
		"grace@example.com", "grace-power-2024", "Grace Hopper")
	if err == nil {
		t.Fatal("expected personal-data rejection, got nil")
	}
	if got := err.Error(); got != errPasswordPersonalData.Error() {
		t.Errorf("expected personal-data error, got: %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- CreateAccount: dictionary password rejected ---

func TestCreateAccount_DictionaryRejected(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	mock.ExpectQuery(`SELECT auth_config FROM projects WHERE`).
		WithArgs(testProjectID).
		WillReturnRows(securityConfigRows(`{"passwordDictionary":true}`))

	// "password" is in the blocklist and is exactly the 8-char minimum.
	_, err := svc.CreateAccount(context.Background(), testProjectID, "unique()",
		"newuser@example.com", "password", "New User")
	if err == nil {
		t.Fatal("expected dictionary rejection, got nil")
	}
	if got := err.Error(); got != errPasswordInDictionary.Error() {
		t.Errorf("expected dictionary error, got: %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- UpdatePassword: reused password rejected within history window ---

func TestUpdatePassword_ReusedRejected(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	oldPassword := "current-pass-9"
	candidate := "reused-pass-1"
	currentHash, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), 4)
	reusedHash, _ := bcrypt.GenerateFromPassword([]byte(candidate), 4)

	// history policy on
	mock.ExpectQuery(`SELECT auth_config FROM projects WHERE`).
		WithArgs(testProjectID).
		WillReturnRows(securityConfigRows(`{"passwordHistory":3}`))

	// fetch current hash to verify old password
	mock.ExpectQuery(`SELECT password_hash FROM users WHERE`).
		WithArgs(testUserID, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"password_hash"}).AddRow(string(currentHash)))

	// history lookup returns a hash matching the candidate
	mock.ExpectQuery(`SELECT password_hash FROM password_history WHERE`).
		WithArgs(testUserID, testProjectID, 3).
		WillReturnRows(sqlmock.NewRows([]string{"password_hash"}).AddRow(string(reusedHash)))

	_, err := svc.UpdatePassword(context.Background(), testUserID, testProjectID, candidate, oldPassword)
	if err == nil {
		t.Fatal("expected reuse rejection, got nil")
	}
	if got := err.Error(); got != errPasswordReused.Error() {
		t.Errorf("expected reuse error, got: %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- UpdatePassword: a fresh password passes the history check ---

func TestUpdatePassword_FreshAccepted(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	oldPassword := "current-pass-9"
	candidate := "brand-new-pass-7"
	currentHash, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), 4)
	otherHash, _ := bcrypt.GenerateFromPassword([]byte("some-older-pass"), 4)

	mock.ExpectQuery(`SELECT auth_config FROM projects WHERE`).
		WithArgs(testProjectID).
		WillReturnRows(securityConfigRows(`{"passwordHistory":3}`))

	mock.ExpectQuery(`SELECT password_hash FROM users WHERE`).
		WithArgs(testUserID, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"password_hash"}).AddRow(string(currentHash)))

	// history has only unrelated hashes, so the candidate is fresh
	mock.ExpectQuery(`SELECT password_hash FROM password_history WHERE`).
		WithArgs(testUserID, testProjectID, 3).
		WillReturnRows(sqlmock.NewRows([]string{"password_hash"}).AddRow(string(otherHash)))

	// the password is written
	mock.ExpectExec(`UPDATE users SET password_hash`).
		WithArgs(sqlmock.AnyArg(), testUserID, testProjectID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// GetAccount reads the user back (best-effort history/session/log writes in
	// between are swallowed by the mock, matching the existing test pattern)
	mock.ExpectQuery(`SELECT .+ FROM users WHERE`).
		WithArgs(testUserID, testProjectID).
		WillReturnRows(userRow(testUserID, testProjectID, testEmail, testName))

	user, err := svc.UpdatePassword(context.Background(), testUserID, testProjectID, candidate, oldPassword)
	if err != nil {
		t.Fatalf("expected fresh password to be accepted, got: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
