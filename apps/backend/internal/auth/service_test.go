package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"database/sql"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mittolabs/applad/internal/db"
	"golang.org/x/crypto/bcrypt"
)

const (
	testProjectID = "proj123"
	testUserID    = "user456"
	testEmail     = "test@example.com"
	testPassword  = "password123"
	testName      = "Test User"
	testJWTSecret = "test-jwt-secret"
)

var testNow = time.Now().UTC().Truncate(time.Second)

func newTestService(t *testing.T) (*Service, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	database := &db.DB{DB: mockDB}
	svc := NewService(database, testJWTSecret)
	return svc, mock, mockDB
}

func userColumns() []string {
	return []string{
		"id", "project_id", "email", "phone", "name",
		"email_verified", "phone_verified", "status",
		"labels", "prefs", "created_at", "updated_at",
	}
}

func userRow(id, projectID, email, name string) *sqlmock.Rows {
	return sqlmock.NewRows(userColumns()).AddRow(
		id, projectID, email, "", name,
		false, false, true,
		[]byte(`[]`), []byte(`{}`), testNow, testNow,
	)
}

// --- 1. TestCreateAccount_Success ---

func TestCreateAccount_Success(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	// Expect INSERT for new user
	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(
			sqlmock.AnyArg(), testProjectID, testEmail, testName,
			sqlmock.AnyArg(), // bcrypt hash
			sqlmock.AnyArg(), // labels JSON
			sqlmock.AnyArg(), // prefs JSON
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect SELECT to fetch created user (GetAccount call)
	mock.ExpectQuery(`SELECT .+ FROM users WHERE`).
		WithArgs(sqlmock.AnyArg(), testProjectID).
		WillReturnRows(userRow(testUserID, testProjectID, testEmail, testName))

	user, err := svc.CreateAccount(context.Background(), testProjectID, "unique()", testEmail, testPassword, testName)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if user.Email != testEmail {
		t.Errorf("expected email %q, got %q", testEmail, user.Email)
	}
	if user.Name != testName {
		t.Errorf("expected name %q, got %q", testName, user.Name)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- 2. TestCreateAccount_DuplicateEmail ---

func TestCreateAccount_DuplicateEmail(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(
			sqlmock.AnyArg(), testProjectID, testEmail, testName,
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnError(fmt.Errorf("Duplicate entry '%s' for key 'users.email'", testEmail))

	_, err := svc.CreateAccount(context.Background(), testProjectID, "unique()", testEmail, testPassword, testName)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "user_already_exists: email already in use" {
		t.Errorf("expected user_already_exists error, got: %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- 3. TestCreateEmailSession_Success ---

func TestCreateEmailSession_Success(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	hash, _ := bcrypt.GenerateFromPassword([]byte(testPassword), 12)

	// SELECT for email login
	mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE email`).
		WithArgs(testEmail, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(testUserID, string(hash)))

	// INSERT session
	mock.ExpectExec(`INSERT INTO sessions`).
		WithArgs(
			sqlmock.AnyArg(), testUserID, testProjectID,
			"127.0.0.1", "TestAgent",
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	res, err := svc.CreateEmailSession(context.Background(), testProjectID, testEmail, testPassword, "", "127.0.0.1", "TestAgent")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Session == nil {
		t.Fatal("expected session, got nil")
	}
	if res.MFAChallenge {
		t.Error("did not expect an MFA challenge for a user with no factor")
	}
	if res.MFAEnrollmentRequired {
		t.Error("did not expect enrollment-required when mfaRequired is off")
	}
	sess, token := res.Session, res.Token
	if sess.UserID != testUserID {
		t.Errorf("expected userID %q, got %q", testUserID, sess.UserID)
	}
	if sess.Provider != "email" {
		t.Errorf("expected provider 'email', got %q", sess.Provider)
	}
	if token == "" {
		t.Error("expected non-empty JWT token")
	}

	// Verify the JWT is valid
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse JWT: %v", err)
	}
	claims := parsed.Claims.(*Claims)
	if claims.Subject != testUserID {
		t.Errorf("JWT sub: expected %q, got %q", testUserID, claims.Subject)
	}
	if claims.ProjectID != testProjectID {
		t.Errorf("JWT pid: expected %q, got %q", testProjectID, claims.ProjectID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- 4. TestCreateEmailSession_InvalidPassword ---

func TestCreateEmailSession_InvalidPassword(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), 12)

	mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE email`).
		WithArgs(testEmail, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(testUserID, string(hash)))

	_, err := svc.CreateEmailSession(context.Background(), testProjectID, testEmail, "wrong-password", "", "127.0.0.1", "TestAgent")
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
	if got := err.Error(); got != "auth: invalid credentials" {
		t.Errorf("expected 'auth: invalid credentials', got: %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- 5. TestCreateEmailSession_UserNotFound ---

func TestCreateEmailSession_UserNotFound(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE email`).
		WithArgs(testEmail, testProjectID).
		WillReturnError(sql.ErrNoRows)

	_, err := svc.CreateEmailSession(context.Background(), testProjectID, testEmail, testPassword, "", "127.0.0.1", "TestAgent")
	if err == nil {
		t.Fatal("expected error for missing user, got nil")
	}
	if got := err.Error(); got != "auth: invalid credentials" {
		t.Errorf("expected 'auth: invalid credentials', got: %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- 6. TestCreateAnonymousSession_Success ---

func TestCreateAnonymousSession_Success(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	// INSERT anonymous user (id, project_id, email, name, labels, prefs, created_at, updated_at)
	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(
			sqlmock.AnyArg(), testProjectID,
			sqlmock.AnyArg(), "Anonymous User",
			sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// INSERT session
	mock.ExpectExec(`INSERT INTO sessions`).
		WithArgs(
			sqlmock.AnyArg(), sqlmock.AnyArg(), testProjectID,
			"10.0.0.1", "AnonBrowser",
			sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	sess, token, err := svc.CreateAnonymousSession(context.Background(), testProjectID, "10.0.0.1", "AnonBrowser")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if sess == nil {
		t.Fatal("expected session, got nil")
	}
	if sess.Provider != "anonymous" {
		t.Errorf("expected provider 'anonymous', got %q", sess.Provider)
	}
	if token == "" {
		t.Error("expected non-empty JWT token")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- 7. TestGetAccount_Success ---

func TestGetAccount_Success(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	mock.ExpectQuery(`SELECT .+ FROM users WHERE`).
		WithArgs(testUserID, testProjectID).
		WillReturnRows(userRow(testUserID, testProjectID, testEmail, testName))

	user, err := svc.GetAccount(context.Background(), testUserID, testProjectID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if user.ID != testUserID {
		t.Errorf("expected ID %q, got %q", testUserID, user.ID)
	}
	if user.Email != testEmail {
		t.Errorf("expected email %q, got %q", testEmail, user.Email)
	}
	if user.Name != testName {
		t.Errorf("expected name %q, got %q", testName, user.Name)
	}
	if user.Labels == nil {
		t.Error("expected labels to be initialized, got nil")
	}
	if user.Prefs == nil {
		t.Error("expected prefs to be initialized, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- 8. TestGetAccount_NotFound ---

func TestGetAccount_NotFound(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	mock.ExpectQuery(`SELECT .+ FROM users WHERE`).
		WithArgs(testUserID, testProjectID).
		WillReturnError(sql.ErrNoRows)

	_, err := svc.GetAccount(context.Background(), testUserID, testProjectID)
	if err == nil {
		t.Fatal("expected error for missing user, got nil")
	}
	// scanUser wraps sql.ErrNoRows as "user not found"
	if got := err.Error(); got != "user not found" {
		t.Errorf("expected 'user not found', got: %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- 9. TestUpdatePassword_VerifiesOldPassword ---

func TestUpdatePassword_VerifiesOldPassword(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	oldPassword := "old-password"
	hash, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), 12)

	// SELECT password_hash
	mock.ExpectQuery(`SELECT password_hash FROM users WHERE`).
		WithArgs(testUserID, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"password_hash"}).AddRow(string(hash)))

	// With wrong old password, should fail
	_, err := svc.UpdatePassword(context.Background(), testUserID, testProjectID, "new-password", "wrong-old-password")
	if err == nil {
		t.Fatal("expected error for wrong old password, got nil")
	}
	if got := err.Error(); got != "auth: invalid old password" {
		t.Errorf("expected 'auth: invalid old password', got: %v", got)
	}

	// Now test with correct old password
	mock.ExpectQuery(`SELECT password_hash FROM users WHERE`).
		WithArgs(testUserID, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"password_hash"}).AddRow(string(hash)))

	mock.ExpectExec(`UPDATE users SET password_hash`).
		WithArgs(sqlmock.AnyArg(), testUserID, testProjectID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(`SELECT .+ FROM users WHERE`).
		WithArgs(testUserID, testProjectID).
		WillReturnRows(userRow(testUserID, testProjectID, testEmail, testName))

	user, err := svc.UpdatePassword(context.Background(), testUserID, testProjectID, "new-password", oldPassword)
	if err != nil {
		t.Fatalf("expected no error with correct old password, got: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- 10. TestSignJWT_ValidToken ---

func TestSignJWT_ValidToken(t *testing.T) {
	svc, _, mockDB := newTestService(t)
	defer mockDB.Close()

	expires := time.Now().UTC().Add(15 * time.Minute)
	token, err := svc.signJWT(testUserID, "sess789", testProjectID, expires)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Parse and verify the token
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(testJWTSecret), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("token is not valid")
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok {
		t.Fatal("failed to cast claims")
	}
	if claims.Subject != testUserID {
		t.Errorf("expected sub %q, got %q", testUserID, claims.Subject)
	}
	if claims.SessionID != "sess789" {
		t.Errorf("expected sid %q, got %q", "sess789", claims.SessionID)
	}
	if claims.ProjectID != testProjectID {
		t.Errorf("expected pid %q, got %q", testProjectID, claims.ProjectID)
	}
	if claims.ID != "sess789" {
		t.Errorf("expected jti %q, got %q", "sess789", claims.ID)
	}

	// Verify it rejects with wrong secret
	_, err = jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte("wrong-secret"), nil
	})
	if err == nil {
		t.Error("expected error when parsing with wrong secret")
	}
}

// --- 11. TestEnableMFA_GeneratesSecret ---

func TestEnableMFA_GeneratesSecret(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	mock.ExpectExec(`UPDATE users SET mfa_secret`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), testUserID, testProjectID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	secret, recovery, err := svc.EnableMFA(context.Background(), testUserID, testProjectID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify secret is valid base32 (no padding)
	if secret == "" {
		t.Fatal("expected non-empty secret")
	}
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		t.Fatalf("secret is not valid base32: %v", err)
	}
	if len(decoded) != 20 {
		t.Errorf("expected 20-byte secret, got %d bytes", len(decoded))
	}

	// Verify 8 recovery codes
	if len(recovery) != 8 {
		t.Fatalf("expected 8 recovery codes, got %d", len(recovery))
	}
	for i, code := range recovery {
		if len(code) != 8 {
			t.Errorf("recovery code %d: expected 8 digits, got %d chars: %q", i, len(code), code)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- 12. TestValidateTOTP ---

func TestValidateTOTP(t *testing.T) {
	// Generate a known secret
	secretBytes := make([]byte, 20)
	for i := range secretBytes {
		secretBytes[i] = byte(i + 1) // deterministic secret
	}
	secretB32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)

	// Generate a valid TOTP code for the current time step
	counter := time.Now().Unix() / 30
	validCode := testGenerateTOTP(secretBytes, counter)

	// Should validate with correct code
	if !validateTOTP(secretB32, validCode) {
		t.Errorf("expected TOTP to validate for code %q", validCode)
	}

	// Should reject wrong code
	if validateTOTP(secretB32, "000000") {
		// Edge case: 000000 could be a valid code, so skip if it matches
		if validCode != "000000" {
			t.Error("expected TOTP to reject wrong code '000000'")
		}
	}

	// Should reject invalid base32 secret
	if validateTOTP("!!!invalid!!!", "123456") {
		t.Error("expected TOTP to reject invalid base32 secret")
	}
}

// testGenerateTOTP mirrors the production generateTOTP for verification.
func testGenerateTOTP(secret []byte, counter int64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))
	mac := hmac.New(sha1.New, secret)
	mac.Write(buf)
	hash := mac.Sum(nil)
	offset := hash[len(hash)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", truncated%1000000)
}

// --- 13. TestCreateAuthToken_Success ---

func TestCreateAuthToken_Success(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	mock.ExpectExec(`INSERT INTO auth_tokens`).
		WithArgs(
			sqlmock.AnyArg(), testUserID, testProjectID,
			"email_verification",
			sqlmock.AnyArg(), // token secret
			sqlmock.AnyArg(), // expires_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	token, err := svc.CreateAuthToken(context.Background(), testUserID, testProjectID, "email_verification", 24*time.Hour)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	// Verify token is valid base32
	_, err = base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(token)
	if err != nil {
		t.Errorf("token is not valid base32: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- 14. TestValidateAuthToken_Expired ---

func TestValidateAuthToken_Expired(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	expiredTime := time.Now().UTC().Add(-1 * time.Hour) // expired 1 hour ago
	tokenSecret := "SOME_TOKEN_SECRET"
	tokenID := "tok123"

	mock.ExpectQuery(`SELECT id, user_id, expires_at FROM auth_tokens WHERE`).
		WithArgs(testProjectID, "password_reset", tokenSecret).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expires_at"}).
			AddRow(tokenID, testUserID, expiredTime))

	// Expect DELETE for expired token cleanup
	mock.ExpectExec(`DELETE FROM auth_tokens WHERE id`).
		WithArgs(tokenID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	_, err := svc.ValidateAuthToken(context.Background(), testProjectID, "password_reset", tokenSecret)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
	if got := err.Error(); got != "auth: token expired" {
		t.Errorf("expected 'auth: token expired', got: %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- 15. TestCreatePasswordResetToken ---

func TestCreatePasswordResetToken(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	// SELECT user by email
	mock.ExpectQuery(`SELECT id FROM users WHERE email`).
		WithArgs(testEmail, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(testUserID))

	// INSERT auth_token
	mock.ExpectExec(`INSERT INTO auth_tokens`).
		WithArgs(
			sqlmock.AnyArg(), testUserID, testProjectID,
			"password_reset",
			sqlmock.AnyArg(), // token
			sqlmock.AnyArg(), // expires_at
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	token, err := svc.CreatePasswordResetToken(context.Background(), testProjectID, testEmail)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// --- MFA enforcement on email sign-in ---

// authConfigRows builds a projects.auth_config row for loadSecurity.
func authConfigRows(mfaRequired bool) *sqlmock.Rows {
	cfg := fmt.Sprintf(`{"security":{"mfaRequired":%t}}`, mfaRequired)
	return sqlmock.NewRows([]string{"auth_config"}).AddRow(cfg)
}

// knownTOTP returns a base32 secret and a currently-valid code for it.
func knownTOTP() (secretB32, code string) {
	secretBytes := make([]byte, 20)
	for i := range secretBytes {
		secretBytes[i] = byte(i + 7)
	}
	secretB32 = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)
	code = testGenerateTOTP(secretBytes, time.Now().Unix()/30)
	return
}

// TestCreateEmailSession_MFAEnrolled_ChallengeRequired: an enrolled user who
// supplies no code is challenged, and no session is opened.
func TestCreateEmailSession_MFAEnrolled_ChallengeRequired(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	hash, _ := bcrypt.GenerateFromPassword([]byte(testPassword), 12)
	mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE email`).
		WithArgs(testEmail, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(testUserID, string(hash)))
	mock.ExpectQuery(`SELECT auth_config FROM projects WHERE`).
		WithArgs(testProjectID).
		WillReturnRows(authConfigRows(false))
	mock.ExpectQuery(`SELECT mfa_enabled FROM users WHERE id`).
		WithArgs(testUserID, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"mfa_enabled"}).AddRow(true))

	res, err := svc.CreateEmailSession(context.Background(), testProjectID, testEmail, testPassword, "", "127.0.0.1", "TestAgent")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !res.MFAChallenge {
		t.Fatal("expected MFAChallenge to be true for an enrolled user with no code")
	}
	if res.Session != nil {
		t.Fatal("expected no session to be opened during an MFA challenge")
	}
}

// TestCreateEmailSession_MFAEnrolled_ValidCode: an enrolled user who supplies a
// valid TOTP code is signed in.
func TestCreateEmailSession_MFAEnrolled_ValidCode(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	secretB32, code := knownTOTP()
	hash, _ := bcrypt.GenerateFromPassword([]byte(testPassword), 12)

	mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE email`).
		WithArgs(testEmail, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(testUserID, string(hash)))
	mock.ExpectQuery(`SELECT auth_config FROM projects WHERE`).
		WithArgs(testProjectID).
		WillReturnRows(authConfigRows(false))
	mock.ExpectQuery(`SELECT mfa_enabled FROM users WHERE id`).
		WithArgs(testUserID, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"mfa_enabled"}).AddRow(true))
	// ValidateMFAForLogin reads the secret + recovery codes.
	mock.ExpectQuery(`SELECT mfa_secret, mfa_recovery FROM users WHERE email`).
		WithArgs(testEmail, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"mfa_secret", "mfa_recovery"}).AddRow(secretB32, []byte(`[]`)))
	mock.ExpectExec(`INSERT INTO sessions`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	res, err := svc.CreateEmailSession(context.Background(), testProjectID, testEmail, testPassword, code, "127.0.0.1", "TestAgent")
	if err != nil {
		t.Fatalf("expected no error with a valid MFA code, got: %v", err)
	}
	if res.MFAChallenge {
		t.Fatal("did not expect a challenge when a valid code was supplied")
	}
	if res.Session == nil || res.Token == "" {
		t.Fatal("expected a session to be opened for a valid MFA code")
	}
}

// TestCreateEmailSession_MFAEnrolled_InvalidCode: an enrolled user with a bad
// code is rejected with the stable errMFAInvalidCode.
func TestCreateEmailSession_MFAEnrolled_InvalidCode(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	secretB32, _ := knownTOTP()
	hash, _ := bcrypt.GenerateFromPassword([]byte(testPassword), 12)

	mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE email`).
		WithArgs(testEmail, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(testUserID, string(hash)))
	mock.ExpectQuery(`SELECT auth_config FROM projects WHERE`).
		WithArgs(testProjectID).
		WillReturnRows(authConfigRows(false))
	mock.ExpectQuery(`SELECT mfa_enabled FROM users WHERE id`).
		WithArgs(testUserID, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"mfa_enabled"}).AddRow(true))
	mock.ExpectQuery(`SELECT mfa_secret, mfa_recovery FROM users WHERE email`).
		WithArgs(testEmail, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"mfa_secret", "mfa_recovery"}).AddRow(secretB32, []byte(`[]`)))

	_, err := svc.CreateEmailSession(context.Background(), testProjectID, testEmail, testPassword, "000000", "127.0.0.1", "TestAgent")
	if err == nil {
		t.Fatal("expected an error for an invalid MFA code")
	}
	if !errors.Is(err, errMFAInvalidCode) {
		t.Fatalf("expected errMFAInvalidCode, got: %v", err)
	}
}

// TestCreateEmailSession_MFARequired_NoFactor_EnrollmentSignal: with the project
// requiring MFA, a user with no factor is NOT locked out. A session opens,
// flagged so the client can force enrolment.
func TestCreateEmailSession_MFARequired_NoFactor_EnrollmentSignal(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	hash, _ := bcrypt.GenerateFromPassword([]byte(testPassword), 12)
	mock.ExpectQuery(`SELECT id, password_hash FROM users WHERE email`).
		WithArgs(testEmail, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(testUserID, string(hash)))
	mock.ExpectQuery(`SELECT auth_config FROM projects WHERE`).
		WithArgs(testProjectID).
		WillReturnRows(authConfigRows(true)) // mfaRequired = true
	mock.ExpectQuery(`SELECT mfa_enabled FROM users WHERE id`).
		WithArgs(testUserID, testProjectID).
		WillReturnRows(sqlmock.NewRows([]string{"mfa_enabled"}).AddRow(false))
	mock.ExpectExec(`INSERT INTO sessions`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	res, err := svc.CreateEmailSession(context.Background(), testProjectID, testEmail, testPassword, "", "127.0.0.1", "TestAgent")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.Session == nil {
		t.Fatal("expected a session (a user with no factor must not be locked out)")
	}
	if res.MFAChallenge {
		t.Fatal("did not expect a challenge for a user with no enrolled factor")
	}
	if !res.MFAEnrollmentRequired {
		t.Fatal("expected MFAEnrollmentRequired to be true when the project requires MFA")
	}
}

// --- 16. TestResetPassword_Success ---

func TestResetPassword_Success(t *testing.T) {
	svc, mock, mockDB := newTestService(t)
	defer mockDB.Close()

	tokenSecret := "VALID_RESET_TOKEN"
	tokenID := "tok456"
	validExpiry := time.Now().UTC().Add(30 * time.Minute) // still valid

	// SELECT auth_token (ValidateAuthToken)
	mock.ExpectQuery(`SELECT id, user_id, expires_at FROM auth_tokens WHERE`).
		WithArgs(testProjectID, "password_reset", tokenSecret).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expires_at"}).
			AddRow(tokenID, testUserID, validExpiry))

	// DELETE consumed token
	mock.ExpectExec(`DELETE FROM auth_tokens WHERE id`).
		WithArgs(tokenID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// UPDATE password
	mock.ExpectExec(`UPDATE users SET password_hash`).
		WithArgs(sqlmock.AnyArg(), testUserID, testProjectID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := svc.ResetPassword(context.Background(), testProjectID, tokenSecret, "new-secure-password")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
