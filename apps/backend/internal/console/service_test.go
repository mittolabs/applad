package console

import (
	"context"
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/totp"
	"golang.org/x/crypto/bcrypt"
)

func newMockService(t *testing.T) (*Service, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	database := &db.DB{DB: mockDB}
	svc := NewService(database, "test-jwt-secret-32chars-long!!")
	return svc, mock
}

// --- JWT tests (no DB needed) ---

func TestSignJWT_Roundtrip(t *testing.T) {
	svc := &Service{jwtSecret: "test-secret-key-12345"}
	token, err := svc.signJWT("user123", "test@example.com", "")
	if err != nil {
		t.Fatalf("signJWT failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	userID, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if userID != "user123" {
		t.Errorf("expected user123, got %s", userID)
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	svc := &Service{jwtSecret: "test-secret"}
	_, err := svc.ValidateToken("invalid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	svc1 := &Service{jwtSecret: "secret-1"}
	svc2 := &Service{jwtSecret: "secret-2"}
	token, _ := svc1.signJWT("user1", "test@test.com", "")
	_, err := svc2.ValidateToken(token)
	if err == nil {
		t.Error("expected error for wrong secret")
	}
}

func TestValidateToken_ConsoleFlag(t *testing.T) {
	svc := &Service{jwtSecret: "test-secret"}
	token, _ := svc.signJWT("user1", "test@test.com", "")
	userID, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("expected valid console token: %v", err)
	}
	if userID != "user1" {
		t.Errorf("expected user1, got %s", userID)
	}
}

// --- SignupEnabled ---

func TestSignupEnabled_True(t *testing.T) {
	svc := &Service{jwtSecret: "test"}
	enabled, err := svc.SignupEnabled(nil, "true")
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("expected true")
	}
}

func TestSignupEnabled_False(t *testing.T) {
	svc := &Service{jwtSecret: "test"}
	enabled, err := svc.SignupEnabled(nil, "false")
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Error("expected false")
	}
}

func TestSignupEnabled_Auto_NoUsers(t *testing.T) {
	svc, mock := newMockService(t)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	enabled, err := svc.SignupEnabled(context.Background(), "auto")
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Error("expected signup enabled when 0 users")
	}
}

func TestSignupEnabled_Auto_HasUsers(t *testing.T) {
	svc, mock := newMockService(t)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	enabled, err := svc.SignupEnabled(context.Background(), "auto")
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Error("expected signup disabled when users exist")
	}
}

// --- Signup with DB ---

func TestSignup_Success(t *testing.T) {
	svc, mock := newMockService(t)
	mock.ExpectExec("INSERT INTO console_users").WillReturnResult(sqlmock.NewResult(1, 1))

	user, token, err := svc.Signup(context.Background(), "admin@test.com", "password123", "Admin")
	if err != nil {
		t.Fatalf("signup failed: %v", err)
	}
	if user.Email != "admin@test.com" {
		t.Errorf("expected admin@test.com, got %s", user.Email)
	}
	if user.Name != "Admin" {
		t.Errorf("expected Admin, got %s", user.Name)
	}
	if user.ID == "" {
		t.Error("expected non-empty ID")
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
	// Verify token is valid
	uid, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("returned token is invalid: %v", err)
	}
	if uid != user.ID {
		t.Errorf("token user ID %s != user ID %s", uid, user.ID)
	}
}

func TestSignup_DuplicateEmail(t *testing.T) {
	svc, mock := newMockService(t)
	mock.ExpectExec("INSERT INTO console_users").WillReturnError(
		fmt.Errorf("Error 1062: Duplicate entry 'admin@test.com' for key 'email'"))

	_, _, err := svc.Signup(context.Background(), "admin@test.com", "password123", "Admin")
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

// --- Login with DB ---

// loginRows builds the row set Login now scans, including the MFA columns.
func loginRows(id, email, name, hash string, mfaEnabled bool, secret, recovery string) *sqlmock.Rows {
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	var sec, rec interface{}
	if secret != "" {
		sec = secret
	}
	if recovery != "" {
		rec = recovery
	}
	return sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "mfa_enabled", "mfa_secret", "mfa_recovery", "created_at", "updated_at"}).
		AddRow(id, email, name, hash, mfaEnabled, sec, rec, ts, ts)
}

func TestLogin_Success(t *testing.T) {
	svc, mock := newMockService(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)

	mock.ExpectQuery("SELECT .+ FROM console_users WHERE email").
		WithArgs("admin@test.com").
		WillReturnRows(loginRows("uid1", "admin@test.com", "Admin", string(hash), false, "", ""))

	user, err := svc.Login(context.Background(), "admin@test.com", "password123", "")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if user.Email != "admin@test.com" {
		t.Errorf("expected admin@test.com, got %s", user.Email)
	}
	if user.MFAEnabled {
		t.Error("expected MFA disabled for this user")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, mock := newMockService(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), 12)

	mock.ExpectQuery("SELECT .+ FROM console_users WHERE email").
		WithArgs("admin@test.com").
		WillReturnRows(loginRows("uid1", "admin@test.com", "Admin", string(hash), false, "", ""))

	_, err := svc.Login(context.Background(), "admin@test.com", "wrong-password", "")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, mock := newMockService(t)
	mock.ExpectQuery("SELECT .+ FROM console_users WHERE email").
		WithArgs("nobody@test.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "mfa_enabled", "mfa_secret", "mfa_recovery", "created_at", "updated_at"}))

	_, err := svc.Login(context.Background(), "nobody@test.com", "password", "")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

// --- GetMe with DB ---

func TestGetMe_Success(t *testing.T) {
	svc, mock := newMockService(t)
	mock.ExpectQuery("SELECT .+ FROM console_users WHERE id").
		WithArgs("uid1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "mfa_enabled", "created_at", "updated_at"}).
			AddRow("uid1", "admin@test.com", "Admin", false, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))

	user, err := svc.GetMe(context.Background(), "uid1")
	if err != nil {
		t.Fatalf("getMe failed: %v", err)
	}
	if user.ID != "uid1" {
		t.Errorf("expected uid1, got %s", user.ID)
	}
}

func TestGetMe_NotFound(t *testing.T) {
	svc, mock := newMockService(t)
	mock.ExpectQuery("SELECT .+ FROM console_users WHERE id").
		WithArgs("missing").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "mfa_enabled", "created_at", "updated_at"}))

	_, err := svc.GetMe(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- CountUsers ---

func TestCountUsers(t *testing.T) {
	svc, mock := newMockService(t)
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	count, err := svc.CountUsers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if count != 5 {
		t.Errorf("expected 5, got %d", count)
	}
}

// --- Console MFA (TOTP) ---

// currentTOTP returns the code an authenticator would show right now for a
// secret, so a test can prove a real code is accepted.
func currentTOTP(secretB32 string) string {
	secret, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secretB32)
	return totp.Generate(secret, time.Now().Unix()/30)
}

// mfaLookupRows builds the row set BeginMFAEnrollment reads before deciding
// whether a re-enrolment needs a current code.
func mfaLookupRows(email string, enabled bool, secret, recovery string) *sqlmock.Rows {
	var sec, rec interface{}
	if secret != "" {
		sec = secret
	}
	if recovery != "" {
		rec = recovery
	}
	return sqlmock.NewRows([]string{"email", "mfa_enabled", "mfa_secret", "mfa_recovery"}).
		AddRow(email, enabled, sec, rec)
}

// BeginMFAEnrollment on an account without MFA persists a secret and returns
// recovery codes, without yet enabling MFA — enrolment completes only once a
// code is verified.
func TestBeginMFAEnrollment_ReturnsSecret(t *testing.T) {
	svc, mock := newMockService(t)
	mock.ExpectQuery("SELECT email, mfa_enabled, mfa_secret, mfa_recovery FROM console_users WHERE id").
		WithArgs("uid1").
		WillReturnRows(mfaLookupRows("admin@test.com", false, "", ""))
	mock.ExpectExec("UPDATE console_users SET mfa_secret = \\$1, mfa_recovery = \\$2, mfa_enabled = FALSE").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "uid1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	secret, uri, recovery, err := svc.BeginMFAEnrollment(context.Background(), "uid1", "")
	if err != nil {
		t.Fatalf("begin enrolment: %v", err)
	}
	if secret == "" {
		t.Error("expected a non-empty secret")
	}
	if len(recovery) != 8 {
		t.Errorf("expected 8 recovery codes, got %d", len(recovery))
	}
	if !strings.HasPrefix(uri, "otpauth://totp/") || !strings.Contains(uri, "secret="+secret) {
		t.Errorf("unexpected otpauth uri: %q", uri)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

// Re-enrolling on an already-enrolled account WITHOUT a valid code must be
// rejected: no write happens, so the live secret is untouched and MFA stays
// enabled. This is the downgrade attack — a walk-up or stolen JWT must not be
// able to swap the second factor.
func TestBeginMFAEnrollment_ReenrollWithoutCodeRejected(t *testing.T) {
	svc, mock := newMockService(t)
	secret, _ := totp.NewSecret()
	mock.ExpectQuery("SELECT email, mfa_enabled, mfa_secret, mfa_recovery FROM console_users WHERE id").
		WithArgs("uid1").
		WillReturnRows(mfaLookupRows("admin@test.com", true, secret, ""))
	// No UPDATE expected — a missing/invalid code must not reach a write.

	_, _, _, err := svc.BeginMFAEnrollment(context.Background(), "uid1", "000000")
	if !errors.Is(err, ErrMFAInvalid) {
		t.Fatalf("expected ErrMFAInvalid, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

// Re-enrolling WITH a valid current code proceeds and stores a new secret,
// while leaving mfa_enabled true (the UPDATE does not clear it).
func TestBeginMFAEnrollment_ReenrollWithValidCodeProceeds(t *testing.T) {
	svc, mock := newMockService(t)
	secret, _ := totp.NewSecret()
	mock.ExpectQuery("SELECT email, mfa_enabled, mfa_secret, mfa_recovery FROM console_users WHERE id").
		WithArgs("uid1").
		WillReturnRows(mfaLookupRows("admin@test.com", true, secret, ""))
	// The update must not mention mfa_enabled — the account stays protected.
	mock.ExpectExec("UPDATE console_users SET mfa_secret = \\$1, mfa_recovery = \\$2 WHERE id = \\$3").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "uid1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	newSecret, _, _, err := svc.BeginMFAEnrollment(context.Background(), "uid1", currentTOTP(secret))
	if err != nil {
		t.Fatalf("re-enrol with valid code: %v", err)
	}
	if newSecret == "" || newSecret == secret {
		t.Error("expected a fresh secret on re-enrolment")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

// A valid code enables MFA; the stored secret is what the code is checked
// against.
func TestVerifyMFA_ValidCodeEnables(t *testing.T) {
	svc, mock := newMockService(t)
	secret, _ := totp.NewSecret()

	mock.ExpectQuery("SELECT mfa_secret FROM console_users WHERE id").
		WithArgs("uid1").
		WillReturnRows(sqlmock.NewRows([]string{"mfa_secret"}).AddRow(secret))
	mock.ExpectExec("UPDATE console_users SET mfa_enabled = TRUE").
		WithArgs("uid1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := svc.VerifyMFA(context.Background(), "uid1", currentTOTP(secret)); err != nil {
		t.Fatalf("verify with valid code: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

// A wrong code must not enable MFA.
func TestVerifyMFA_InvalidCodeRejected(t *testing.T) {
	svc, mock := newMockService(t)
	secret, _ := totp.NewSecret()

	mock.ExpectQuery("SELECT mfa_secret FROM console_users WHERE id").
		WithArgs("uid1").
		WillReturnRows(sqlmock.NewRows([]string{"mfa_secret"}).AddRow(secret))

	if err := svc.VerifyMFA(context.Background(), "uid1", "000000"); !errors.Is(err, ErrMFAInvalid) {
		t.Fatalf("expected ErrMFAInvalid, got %v", err)
	}
}

// When MFA is enabled, a correct password with no code returns ErrMFARequired —
// the sign-in does not complete on the password alone.
func TestLogin_MFARequiredWithoutCode(t *testing.T) {
	svc, mock := newMockService(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	secret, _ := totp.NewSecret()

	mock.ExpectQuery("SELECT .+ FROM console_users WHERE email").
		WithArgs("admin@test.com").
		WillReturnRows(loginRows("uid1", "admin@test.com", "Admin", string(hash), true, secret, ""))

	_, err := svc.Login(context.Background(), "admin@test.com", "password123", "")
	if !errors.Is(err, ErrMFARequired) {
		t.Fatalf("expected ErrMFARequired, got %v", err)
	}
}

// A correct password plus a valid code completes the sign-in.
func TestLogin_MFAValidCode(t *testing.T) {
	svc, mock := newMockService(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	secret, _ := totp.NewSecret()

	mock.ExpectQuery("SELECT .+ FROM console_users WHERE email").
		WithArgs("admin@test.com").
		WillReturnRows(loginRows("uid1", "admin@test.com", "Admin", string(hash), true, secret, ""))

	user, err := svc.Login(context.Background(), "admin@test.com", "password123", currentTOTP(secret))
	if err != nil {
		t.Fatalf("expected success with valid code, got %v", err)
	}
	if user == nil || !user.MFAEnabled {
		t.Error("expected an MFA-enabled user on success")
	}
}

// A correct password with a wrong code is rejected as ErrMFAInvalid, not as a
// completed sign-in.
func TestLogin_MFAInvalidCode(t *testing.T) {
	svc, mock := newMockService(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)
	secret, _ := totp.NewSecret()

	mock.ExpectQuery("SELECT .+ FROM console_users WHERE email").
		WithArgs("admin@test.com").
		WillReturnRows(loginRows("uid1", "admin@test.com", "Admin", string(hash), true, secret, ""))

	_, err := svc.Login(context.Background(), "admin@test.com", "password123", "000000")
	if !errors.Is(err, ErrMFAInvalid) {
		t.Fatalf("expected ErrMFAInvalid, got %v", err)
	}
}

// An invite is redeemed with its token, not by claiming an invited address.
// Knowing who was invited must never be enough to take their seat.
func TestRedeemInviteUsesTokenNotEmail(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("FROM organization_members").WithArgs("bad-token").
		WillReturnError(sql.ErrNoRows)

	if _, _, err := svc.RedeemInvite(context.Background(), "bad-token", "hunter2hunter2", "Someone"); err == nil {
		t.Error("an unknown token must not create an account")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

// The address comes from the invite record, so a caller cannot choose which
// account gets created.
func TestLookupInviteReportsExistingAccount(t *testing.T) {
	svc, mock := newMockService(t)

	mock.ExpectQuery("FROM organization_members").WithArgs("tok").
		WillReturnRows(sqlmock.NewRows([]string{"email", "name", "role", "org_id", "name"}).
			AddRow("colleague@example.com", "Colleague", "member", "org1", "Tito's Workspace"))
	mock.ExpectQuery("FROM console_users").WithArgs("colleague@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	inv, err := svc.LookupInvite(context.Background(), "tok")
	if err != nil {
		t.Fatal(err)
	}
	if inv.Email != "colleague@example.com" || inv.OrganizationName != "Tito's Workspace" {
		t.Errorf("unexpected invite: %+v", inv)
	}
	if !inv.HasAccount {
		t.Error("should report that this address already has an account, so it signs in instead")
	}
}
