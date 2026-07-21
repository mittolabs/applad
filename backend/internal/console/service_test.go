package console

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mittolabs/applad/internal/db"
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

func TestLogin_Success(t *testing.T) {
	svc, mock := newMockService(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)

	mock.ExpectQuery("SELECT .+ FROM console_users WHERE email").
		WithArgs("admin@test.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "created_at", "updated_at"}).
			AddRow("uid1", "admin@test.com", "Admin", string(hash), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))

	user, token, err := svc.Login(context.Background(), "admin@test.com", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if user.Email != "admin@test.com" {
		t.Errorf("expected admin@test.com, got %s", user.Email)
	}
	if token == "" {
		t.Error("expected token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, mock := newMockService(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), 12)

	mock.ExpectQuery("SELECT .+ FROM console_users WHERE email").
		WithArgs("admin@test.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "created_at", "updated_at"}).
			AddRow("uid1", "admin@test.com", "Admin", string(hash), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))

	_, _, err := svc.Login(context.Background(), "admin@test.com", "wrong-password")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, mock := newMockService(t)
	mock.ExpectQuery("SELECT .+ FROM console_users WHERE email").
		WithArgs("nobody@test.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "password_hash", "created_at", "updated_at"}))

	_, _, err := svc.Login(context.Background(), "nobody@test.com", "password")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

// --- GetMe with DB ---

func TestGetMe_Success(t *testing.T) {
	svc, mock := newMockService(t)
	mock.ExpectQuery("SELECT .+ FROM console_users WHERE id").
		WithArgs("uid1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "created_at", "updated_at"}).
			AddRow("uid1", "admin@test.com", "Admin", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))

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
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "created_at", "updated_at"}))

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

// Self-hosted instances close signup behind the first account. An invited
// colleague must still be able to register, because accepting an invite
// requires an account to attach it to — otherwise a self-hosted team is
// permanently stuck at one person.
func TestSignupAllowedForInvitedAddress(t *testing.T) {
	svc, mock := newMockService(t)

	// "auto" with an existing account: signup is closed.
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("organization_members").WithArgs("invited@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	allowed, err := svc.SignupAllowedFor(context.Background(), "invited@example.com", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("an invited address must be able to register on a closed instance")
	}

	// Same instance, an address nobody invited.
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("organization_members").WithArgs("stranger@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	allowed, err = svc.SignupAllowedFor(context.Background(), "stranger@example.com", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Error("an uninvited address must not be able to register on a closed instance")
	}
}

// The hosted service runs with signup open, where no invite is needed and no
// invite lookup should happen.
func TestSignupAllowedForOpenInstance(t *testing.T) {
	svc, mock := newMockService(t)

	allowed, err := svc.SignupAllowedFor(context.Background(), "anyone@example.com", "true")
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("open signup must allow any address")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("open signup should not query the database: %v", err)
	}
}
