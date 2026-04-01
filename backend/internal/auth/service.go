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

// UpdateUserEmailAdmin updates a user's email server-side (no password required).
func (s *Service) UpdateUserEmailAdmin(ctx context.Context, userID, projectID, email string) (*model.User, error) {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET email = ? WHERE id = ? AND project_id = ?", email, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("auth: update email admin: %w", err)
	}
	return s.GetAccount(ctx, userID, projectID)
}
