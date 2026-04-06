// Package auth implements Applad's authentication service:
// accounts, sessions, and server-side user management.
package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"math/big"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/model"
	"github.com/mittolabs/applad/internal/uid"
	"golang.org/x/crypto/bcrypt"
)

// Claims is the JWT claims structure.
type Claims struct {
	jwt.RegisteredClaims
	SessionID string `json:"sid"`
	ProjectID string `json:"pid"`
}

// Service handles auth business logic.
type Service struct {
	db        *db.DB
	jwtSecret string
}

// NewService creates a new auth Service.
func NewService(database *db.DB, jwtSecret string) *Service {
	return &Service{db: database, jwtSecret: jwtSecret}
}

// CreateAccount creates a new user account.
func (s *Service) CreateAccount(ctx context.Context, projectID, userID, email, password, name string) (*model.User, error) {
	id := uid.New(userID)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}
	now := time.Now().UTC()
	labelsJSON, _ := json.Marshal([]string{})
	prefsJSON, _ := json.Marshal(map[string]interface{}{})

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO users (id, project_id, email, name, password_hash, labels, prefs, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, projectID, email, name, string(hash), labelsJSON, prefsJSON, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "duplicate") {
			return nil, fmt.Errorf("user_already_exists: email already in use")
		}
		return nil, fmt.Errorf("auth: create account: %w", err)
	}
	return s.GetAccount(ctx, id, projectID)
}

// GetAccount returns the user account by userID.
func (s *Service) GetAccount(ctx context.Context, userID, projectID string) (*model.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, email, phone, name, email_verified, phone_verified, status, labels, prefs, created_at, updated_at
		 FROM users WHERE id = ? AND project_id = ?`, userID, projectID)
	return scanUser(row)
}

// GetUser is an alias for GetAccount (used by server-side user management).
func (s *Service) GetUser(ctx context.Context, userID, projectID string) (*model.User, error) {
	return s.GetAccount(ctx, userID, projectID)
}

// UpdateName updates a user's display name.
func (s *Service) UpdateName(ctx context.Context, userID, projectID, name string) (*model.User, error) {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET name = ? WHERE id = ? AND project_id = ?", name, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("auth: update name: %w", err)
	}
	return s.GetAccount(ctx, userID, projectID)
}

// UpdateEmail updates a user's email after verifying the password.
func (s *Service) UpdateEmail(ctx context.Context, userID, projectID, email, password string) (*model.User, error) {
	var hash string
	err := s.db.QueryRowContext(ctx,
		"SELECT password_hash FROM users WHERE id = ? AND project_id = ?", userID, projectID).Scan(&hash)
	if err != nil {
		return nil, fmt.Errorf("auth: user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, fmt.Errorf("auth: invalid password")
	}
	_, err = s.db.ExecContext(ctx,
		"UPDATE users SET email = ? WHERE id = ? AND project_id = ?", email, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("auth: update email: %w", err)
	}
	return s.GetAccount(ctx, userID, projectID)
}

// UpdatePassword updates a user's password after verifying the old one.
func (s *Service) UpdatePassword(ctx context.Context, userID, projectID, password, oldPassword string) (*model.User, error) {
	var hash string
	err := s.db.QueryRowContext(ctx,
		"SELECT password_hash FROM users WHERE id = ? AND project_id = ?", userID, projectID).Scan(&hash)
	if err != nil {
		return nil, fmt.Errorf("auth: user not found")
	}
	if oldPassword != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(oldPassword)); err != nil {
			return nil, fmt.Errorf("auth: invalid old password")
		}
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx,
		"UPDATE users SET password_hash = ? WHERE id = ? AND project_id = ?", string(newHash), userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("auth: update password: %w", err)
	}
	return s.GetAccount(ctx, userID, projectID)
}

// UpdatePrefs updates a user's preferences.
func (s *Service) UpdatePrefs(ctx context.Context, userID, projectID string, prefs map[string]interface{}) (*model.User, error) {
	prefsJSON, err := json.Marshal(prefs)
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx,
		"UPDATE users SET prefs = ? WHERE id = ? AND project_id = ?", prefsJSON, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("auth: update prefs: %w", err)
	}
	return s.GetAccount(ctx, userID, projectID)
}

// DeleteAccount deletes a user account.
func (s *Service) DeleteAccount(ctx context.Context, userID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM users WHERE id = ? AND project_id = ?", userID, projectID)
	return err
}

// CreateEmailSession creates a session by validating email+password.
func (s *Service) CreateEmailSession(ctx context.Context, projectID, email, password, ip, ua string) (*model.Session, string, error) {
	var userID, hash string
	err := s.db.QueryRowContext(ctx,
		"SELECT id, password_hash FROM users WHERE email = ? AND project_id = ? AND status = 1",
		email, projectID).Scan(&userID, &hash)
	if err != nil {
		return nil, "", fmt.Errorf("auth: invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, "", fmt.Errorf("auth: invalid credentials")
	}
	return s.createSession(ctx, userID, projectID, "email", ip, ua)
}

// CreateAnonymousSession creates an anonymous session.
func (s *Service) CreateAnonymousSession(ctx context.Context, projectID, ip, ua string) (*model.Session, string, error) {
	// Create an anonymous user first
	userID := uid.New("unique()")
	now := time.Now().UTC()
	labelsJSON, _ := json.Marshal([]string{})
	prefsJSON, _ := json.Marshal(map[string]interface{}{})
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, project_id, labels, prefs, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		userID, projectID, labelsJSON, prefsJSON, now, now)
	if err != nil {
		return nil, "", fmt.Errorf("auth: create anon user: %w", err)
	}
	return s.createSession(ctx, userID, projectID, "anonymous", ip, ua)
}

func (s *Service) createSession(ctx context.Context, userID, projectID, provider, ip, ua string) (*model.Session, string, error) {
	sessionID := uid.New("unique()")
	expires := time.Now().UTC().Add(365 * 24 * time.Hour)
	now := time.Now().UTC()

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO sessions (id, user_id, project_id, ip, user_agent, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		sessionID, userID, projectID, ip, ua, expires, now)
	if err != nil {
		return nil, "", fmt.Errorf("auth: create session: %w", err)
	}

	token, err := s.signJWT(userID, sessionID, projectID, expires)
	if err != nil {
		return nil, "", err
	}

	sess := &model.Session{
		ID:        sessionID,
		CreatedAt: now,
		UserID:    userID,
		Expire:    expires,
		Provider:  provider,
		IP:        ip,
		UserAgent: ua,
		Current:   true,
	}
	return sess, token, nil
}

// GetSession returns a single session.
func (s *Service) GetSession(ctx context.Context, sessionID, userID, projectID string) (*model.Session, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, user_id, ip, user_agent, expires_at, created_at FROM sessions WHERE id = ? AND user_id = ? AND project_id = ?",
		sessionID, userID, projectID)
	return scanSession(row)
}

// ListSessions returns all sessions for a user.
func (s *Service) ListSessions(ctx context.Context, userID, projectID string) ([]*model.Session, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, user_id, ip, user_agent, expires_at, created_at FROM sessions WHERE user_id = ? AND project_id = ? ORDER BY created_at DESC",
		userID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

// DeleteSession deletes a single session.
func (s *Service) DeleteSession(ctx context.Context, sessionID, userID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM sessions WHERE id = ? AND user_id = ? AND project_id = ?", sessionID, userID, projectID)
	return err
}

// DeleteSessions deletes all sessions for a user.
func (s *Service) DeleteSessions(ctx context.Context, userID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM sessions WHERE user_id = ? AND project_id = ?", userID, projectID)
	return err
}

// GetJWT creates a short-lived JWT (15 min) for the current user.
func (s *Service) GetJWT(ctx context.Context, userID, projectID string) (string, error) {
	expires := time.Now().UTC().Add(15 * time.Minute)
	return s.signJWT(userID, "", projectID, expires)
}

// CreateUser creates a user server-side (no email verification required).
func (s *Service) CreateUser(ctx context.Context, projectID, userID, email, phone, password, name string) (*model.User, error) {
	id := uid.New(userID)
	now := time.Now().UTC()
	labelsJSON, _ := json.Marshal([]string{})
	prefsJSON, _ := json.Marshal(map[string]interface{}{})

	var hashStr sql.NullString
	if password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
		if err != nil {
			return nil, err
		}
		hashStr = sql.NullString{String: string(hash), Valid: true}
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, project_id, email, phone, name, password_hash, labels, prefs, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, projectID,
		nullString(email), nullString(phone), nullString(name),
		hashStr, labelsJSON, prefsJSON, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "duplicate") {
			return nil, fmt.Errorf("user_already_exists: email already in use")
		}
		return nil, fmt.Errorf("auth: create user: %w", err)
	}
	return s.GetAccount(ctx, id, projectID)
}

// ListUsers returns a paginated list of users for a project.
func (s *Service) ListUsers(ctx context.Context, projectID string, limit, offset int, search string) ([]*model.User, int, error) {
	if limit <= 0 {
		limit = 25
	}
	var args []interface{}
	query := `SELECT id, project_id, email, phone, name, email_verified, phone_verified, status, labels, prefs, created_at, updated_at
	          FROM users WHERE project_id = ?`
	args = append(args, projectID)

	if search != "" {
		query += " AND (name LIKE ? OR email LIKE ?)"
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern)
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	var total int
	countQ := "SELECT COUNT(*) FROM users WHERE project_id = ?"
	countArgs := []interface{}{projectID}
	if search != "" {
		countQ += " AND (name LIKE ? OR email LIKE ?)"
		pattern := "%" + search + "%"
		countArgs = append(countArgs, pattern, pattern)
	}
	s.db.QueryRowContext(ctx, countQ, countArgs...).Scan(&total) //nolint:errcheck
	return users, total, nil
}

// UpdateUserStatus enables/disables a user.
func (s *Service) UpdateUserStatus(ctx context.Context, userID, projectID string, status bool) (*model.User, error) {
	statusInt := 0
	if status {
		statusInt = 1
	}
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET status = ? WHERE id = ? AND project_id = ?", statusInt, userID, projectID)
	if err != nil {
		return nil, err
	}
	return s.GetAccount(ctx, userID, projectID)
}

// DeleteUser deletes a user.
func (s *Service) DeleteUser(ctx context.Context, userID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM users WHERE id = ? AND project_id = ?", userID, projectID)
	return err
}

// ListUserSessions returns all sessions for a user (server-side).
func (s *Service) ListUserSessions(ctx context.Context, userID, projectID string) ([]*model.Session, error) {
	return s.ListSessions(ctx, userID, projectID)
}

// DeleteUserSessions deletes all sessions for a user (server-side).
func (s *Service) DeleteUserSessions(ctx context.Context, userID, projectID string) error {
	return s.DeleteSessions(ctx, userID, projectID)
}

// signJWT creates and signs a JWT.
func (s *Service) signJWT(userID, sessionID, projectID string, expires time.Time) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expires),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        sessionID,
		},
		SessionID: sessionID,
		ProjectID: projectID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// --- MFA (TOTP) ---

// EnableMFA generates a TOTP secret for a user and returns it.
func (s *Service) EnableMFA(ctx context.Context, userID, projectID string) (string, []string, error) {
	// Generate random 20-byte secret
	secret := make([]byte, 20)
	rand.Read(secret)
	secretB32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)

	// Generate 8 recovery codes
	recovery := make([]string, 8)
	for i := range recovery {
		code := make([]byte, 4)
		rand.Read(code)
		n := new(big.Int).SetBytes(code)
		recovery[i] = fmt.Sprintf("%08d", n.Int64()%100000000)
	}
	recoveryJSON, _ := json.Marshal(recovery)

	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET mfa_secret = ?, mfa_recovery = ? WHERE id = ? AND project_id = ?",
		secretB32, recoveryJSON, userID, projectID)
	if err != nil {
		return "", nil, err
	}

	return secretB32, recovery, nil
}

// VerifyMFA verifies a TOTP code and enables MFA if valid.
func (s *Service) VerifyMFA(ctx context.Context, userID, projectID, code string) error {
	var secretB32 string
	err := s.db.QueryRowContext(ctx,
		"SELECT mfa_secret FROM users WHERE id = ? AND project_id = ?",
		userID, projectID).Scan(&secretB32)
	if err != nil || secretB32 == "" {
		return fmt.Errorf("auth: MFA not set up")
	}

	if !validateTOTP(secretB32, code) {
		return fmt.Errorf("auth: invalid MFA code")
	}

	_, err = s.db.ExecContext(ctx,
		"UPDATE users SET mfa_enabled = 1 WHERE id = ? AND project_id = ?",
		userID, projectID)
	return err
}

// DisableMFA disables MFA for a user.
func (s *Service) DisableMFA(ctx context.Context, userID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET mfa_enabled = 0, mfa_secret = NULL, mfa_recovery = NULL WHERE id = ? AND project_id = ?",
		userID, projectID)
	return err
}

// CheckMFA returns whether MFA is enabled for the user with the given email.
func (s *Service) CheckMFA(ctx context.Context, projectID, email string) (bool, error) {
	var enabled bool
	err := s.db.QueryRowContext(ctx,
		"SELECT mfa_enabled FROM users WHERE email = ? AND project_id = ? AND status = 1",
		email, projectID).Scan(&enabled)
	if err != nil {
		return false, err
	}
	return enabled, nil
}

// ValidateMFAForLogin validates a TOTP code during login.
func (s *Service) ValidateMFAForLogin(ctx context.Context, projectID, email, code string) error {
	var secretB32 string
	var recoveryJSON []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT mfa_secret, mfa_recovery FROM users WHERE email = ? AND project_id = ?",
		email, projectID).Scan(&secretB32, &recoveryJSON)
	if err != nil {
		return fmt.Errorf("auth: user not found")
	}

	// Try TOTP first
	if validateTOTP(secretB32, code) {
		return nil
	}

	// Try recovery codes
	var recovery []string
	json.Unmarshal(recoveryJSON, &recovery)
	for i, rc := range recovery {
		if rc == code {
			// Remove used recovery code
			recovery = append(recovery[:i], recovery[i+1:]...)
			newJSON, _ := json.Marshal(recovery)
			s.db.ExecContext(ctx,
				"UPDATE users SET mfa_recovery = ? WHERE email = ? AND project_id = ?",
				newJSON, email, projectID)
			return nil
		}
	}

	return fmt.Errorf("auth: invalid MFA code")
}

func validateTOTP(secretB32, code string) bool {
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secretB32)
	if err != nil {
		return false
	}
	// Check current and adjacent time steps (±1)
	now := time.Now().Unix() / 30
	for _, offset := range []int64{-1, 0, 1} {
		expected := generateTOTP(secret, now+offset)
		if expected == code {
			return true
		}
	}
	return false
}

func generateTOTP(secret []byte, counter int64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))
	mac := hmac.New(sha1.New, secret)
	mac.Write(buf)
	hash := mac.Sum(nil)
	offset := hash[len(hash)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", truncated%1000000)
}

// --- Auth tokens (magic link, email verification, password reset) ---

// CreateAuthToken creates a token for email verification, password reset, or magic link.
func (s *Service) CreateAuthToken(ctx context.Context, userID, projectID, tokenType string, ttl time.Duration) (string, error) {
	id := uid.New("unique()")
	secret := make([]byte, 32)
	rand.Read(secret)
	token := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	expires := time.Now().UTC().Add(ttl)

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO auth_tokens (id, user_id, project_id, type, secret, expires_at) VALUES (?, ?, ?, ?, ?, ?)",
		id, userID, projectID, tokenType, token, expires)
	if err != nil {
		return "", err
	}
	return token, nil
}

// ValidateAuthToken validates and consumes a token.
func (s *Service) ValidateAuthToken(ctx context.Context, projectID, tokenType, secret string) (string, error) {
	var id, userID string
	var expiresAt time.Time
	err := s.db.QueryRowContext(ctx,
		"SELECT id, user_id, expires_at FROM auth_tokens WHERE project_id = ? AND type = ? AND secret = ?",
		projectID, tokenType, secret).Scan(&id, &userID, &expiresAt)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("auth: invalid or expired token")
	}
	if err != nil {
		return "", err
	}
	if time.Now().After(expiresAt) {
		s.db.ExecContext(ctx, "DELETE FROM auth_tokens WHERE id = ?", id)
		return "", fmt.Errorf("auth: token expired")
	}
	// Consume token
	s.db.ExecContext(ctx, "DELETE FROM auth_tokens WHERE id = ?", id)
	return userID, nil
}

// CreateMagicLinkToken creates a magic link token for passwordless login.
func (s *Service) CreateMagicLinkToken(ctx context.Context, projectID, email string) (string, error) {
	var userID string
	err := s.db.QueryRowContext(ctx,
		"SELECT id FROM users WHERE email = ? AND project_id = ?",
		email, projectID).Scan(&userID)
	if err == sql.ErrNoRows {
		// Create user without password
		userID = uid.New("unique()")
		now := time.Now().UTC()
		labelsJSON, _ := json.Marshal([]string{})
		prefsJSON, _ := json.Marshal(map[string]interface{}{})
		_, err = s.db.ExecContext(ctx,
			"INSERT INTO users (id, project_id, email, labels, prefs, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			userID, projectID, email, labelsJSON, prefsJSON, now, now)
		if err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	return s.CreateAuthToken(ctx, userID, projectID, "magic_link", 15*time.Minute)
}

// RedeemMagicLink validates a magic link and creates a session.
func (s *Service) RedeemMagicLink(ctx context.Context, projectID, secret, ip, ua string) (*model.Session, string, error) {
	userID, err := s.ValidateAuthToken(ctx, projectID, "magic_link", secret)
	if err != nil {
		return nil, "", err
	}
	// Mark email as verified
	s.db.ExecContext(ctx, "UPDATE users SET email_verified = 1 WHERE id = ? AND project_id = ?", userID, projectID)
	return s.createSession(ctx, userID, projectID, "magic_link", ip, ua)
}

// CreateEmailVerificationToken creates an email verification token.
func (s *Service) CreateEmailVerificationToken(ctx context.Context, userID, projectID string) (string, error) {
	return s.CreateAuthToken(ctx, userID, projectID, "email_verification", 24*time.Hour)
}

// VerifyEmail validates an email verification token.
func (s *Service) VerifyEmail(ctx context.Context, projectID, secret string) error {
	userID, err := s.ValidateAuthToken(ctx, projectID, "email_verification", secret)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		"UPDATE users SET email_verified = 1 WHERE id = ? AND project_id = ?", userID, projectID)
	return err
}

// CreatePasswordResetToken creates a password reset token.
func (s *Service) CreatePasswordResetToken(ctx context.Context, projectID, email string) (string, error) {
	var userID string
	err := s.db.QueryRowContext(ctx,
		"SELECT id FROM users WHERE email = ? AND project_id = ?",
		email, projectID).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("auth: user not found")
	}
	return s.CreateAuthToken(ctx, userID, projectID, "password_reset", 1*time.Hour)
}

// ResetPassword validates a password reset token and sets a new password.
func (s *Service) ResetPassword(ctx context.Context, projectID, secret, newPassword string) error {
	userID, err := s.ValidateAuthToken(ctx, projectID, "password_reset", secret)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		"UPDATE users SET password_hash = ? WHERE id = ? AND project_id = ?",
		string(hash), userID, projectID)
	return err
}

// --- scan helpers ---

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanUser(row *sql.Row) (*model.User, error) {
	var u model.User
	var email, phone, name sql.NullString
	var labelsJSON, prefsJSON []byte
	err := row.Scan(
		&u.ID, new(string), &email, &phone, &name,
		&u.EmailVerified, &u.PhoneVerified, &u.Status,
		&labelsJSON, &prefsJSON,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}
	u.Email = email.String
	u.Phone = phone.String
	u.Name = name.String
	unmarshalStringSlice(labelsJSON, &u.Labels)
	unmarshalMap(prefsJSON, &u.Prefs)
	if u.Labels == nil {
		u.Labels = []string{}
	}
	if u.Prefs == nil {
		u.Prefs = map[string]interface{}{}
	}
	return &u, nil
}

func scanUserRow(rows *sql.Rows) (*model.User, error) {
	var u model.User
	var email, phone, name sql.NullString
	var labelsJSON, prefsJSON []byte
	err := rows.Scan(
		&u.ID, new(string), &email, &phone, &name,
		&u.EmailVerified, &u.PhoneVerified, &u.Status,
		&labelsJSON, &prefsJSON,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.Email = email.String
	u.Phone = phone.String
	u.Name = name.String
	unmarshalStringSlice(labelsJSON, &u.Labels)
	unmarshalMap(prefsJSON, &u.Prefs)
	if u.Labels == nil {
		u.Labels = []string{}
	}
	if u.Prefs == nil {
		u.Prefs = map[string]interface{}{}
	}
	return &u, nil
}

func scanSession(row *sql.Row) (*model.Session, error) {
	var s model.Session
	var ip, ua sql.NullString
	err := row.Scan(&s.ID, &s.UserID, &ip, &ua, &s.Expire, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, err
	}
	s.IP = ip.String
	s.UserAgent = ua.String
	return &s, nil
}

func scanSessions(rows *sql.Rows) ([]*model.Session, error) {
	var sessions []*model.Session
	for rows.Next() {
		var s model.Session
		var ip, ua sql.NullString
		if err := rows.Scan(&s.ID, &s.UserID, &ip, &ua, &s.Expire, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.IP = ip.String
		s.UserAgent = ua.String
		sessions = append(sessions, &s)
	}
	return sessions, nil
}

func unmarshalStringSlice(data []byte, v *[]string) {
	if len(data) > 0 {
		json.Unmarshal(data, v) //nolint:errcheck
	}
}

func unmarshalMap(data []byte, v *map[string]interface{}) {
	if len(data) > 0 {
		json.Unmarshal(data, v) //nolint:errcheck
	}
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// CreateOAuthSession creates or links an OAuth user and returns a session.
func (s *Service) CreateOAuthSession(ctx context.Context, projectID, provider, oauthID, email, name, ip, ua string) (*model.Session, string, error) {
	// Check if user exists with this OAuth identity
	var userID string
	err := s.db.QueryRowContext(ctx,
		"SELECT id FROM users WHERE project_id = ? AND oauth_provider = ? AND oauth_id = ?",
		projectID, provider, oauthID).Scan(&userID)

	if err == sql.ErrNoRows {
		// Check if user exists with this email
		err2 := s.db.QueryRowContext(ctx,
			"SELECT id FROM users WHERE project_id = ? AND email = ?",
			projectID, email).Scan(&userID)

		if err2 == sql.ErrNoRows {
			// Create new user
			userID = uid.New("unique()")
			now := time.Now().UTC()
			labelsJSON, _ := json.Marshal([]string{})
			prefsJSON, _ := json.Marshal(map[string]interface{}{})
			_, err := s.db.ExecContext(ctx,
				`INSERT INTO users (id, project_id, email, name, oauth_provider, oauth_id, email_verified, labels, prefs, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?)`,
				userID, projectID, email, name, provider, oauthID, labelsJSON, prefsJSON, now, now)
			if err != nil {
				return nil, "", fmt.Errorf("auth: create oauth user: %w", err)
			}
		} else if err2 == nil {
			// Link OAuth to existing email user
			s.db.ExecContext(ctx,
				"UPDATE users SET oauth_provider = ?, oauth_id = ?, email_verified = 1 WHERE id = ? AND project_id = ?",
				provider, oauthID, userID, projectID)
		} else {
			return nil, "", err2
		}
	} else if err != nil {
		return nil, "", err
	}

	return s.createSession(ctx, userID, projectID, provider, ip, ua)
}

// ListOAuthProviders returns which OAuth providers are configured for a project.
func (s *Service) ListOAuthProviders(providers []string) []map[string]interface{} {
	var result []map[string]interface{}
	for _, p := range providers {
		result = append(result, map[string]interface{}{
			"provider": p,
			"enabled":  true,
		})
	}
	return result
}

// UpdateUserEmailAdmin updates a user's email server-side (no password required).
func (s *Service) UpdateUserEmailAdmin(ctx context.Context, userID, projectID, email string) (*model.User, error) {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET email = ? WHERE id = ? AND project_id = ?", email, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("auth: update email admin: %w", err)
	}
	return s.GetAccount(ctx, userID, projectID)
}
