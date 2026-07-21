// Package console implements authentication for the Applad admin console.
// This is separate from per-project user auth — console users are system-level
// administrators who manage projects via the web console.
package console

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
	"golang.org/x/crypto/bcrypt"
)

// resetEntry stores a pending password-reset token.
type resetEntry struct {
	userID    string
	expiresAt time.Time
}

var resetTokens sync.Map // string → resetEntry

// ConsoleUser represents an admin console user.
type ConsoleUser struct {
	ID        string    `json:"$id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"$createdAt"`
	UpdatedAt time.Time `json:"$updatedAt"`
}

// ConsoleClaims is the JWT claims structure for console auth.
type ConsoleClaims struct {
	jwt.RegisteredClaims
	Console   bool   `json:"console"`
	SessionID string `json:"sid,omitempty"`
}

// Service handles console auth business logic.
type Service struct {
	db        *db.DB
	jwtSecret string
}

// NewService creates a new console auth Service.
func NewService(database *db.DB, jwtSecret string) *Service {
	return &Service{db: database, jwtSecret: jwtSecret}
}

// Signup creates a new console admin user.
func (s *Service) Signup(ctx context.Context, email, password, name string) (*ConsoleUser, string, error) {
	id := uid.New("unique()")
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, "", fmt.Errorf("console: hash password: %w", err)
	}
	now := time.Now().UTC()

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO console_users (id, email, name, password_hash, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, email, name, string(hash), now, now)
	if err != nil {
		return nil, "", fmt.Errorf("console: signup: %w", err)
	}

	token, err := s.signJWT(id, email, "")
	if err != nil {
		return nil, "", err
	}

	return &ConsoleUser{
		ID: id, Email: email, Name: name,
		CreatedAt: now, UpdatedAt: now,
	}, token, nil
}

// Login authenticates a console user by email+password.
func (s *Service) Login(ctx context.Context, email, password string) (*ConsoleUser, string, error) {
	var u ConsoleUser
	var hash string
	var name sql.NullString

	err := s.db.QueryRowContext(ctx,
		"SELECT id, email, name, password_hash, created_at, updated_at FROM console_users WHERE email = $1",
		email).Scan(&u.ID, &u.Email, &name, &hash, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, "", fmt.Errorf("console: invalid credentials")
	}
	if err != nil {
		return nil, "", err
	}
	u.Name = name.String

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, "", fmt.Errorf("console: invalid credentials")
	}

	token, err := s.signJWT(u.ID, u.Email, "")
	if err != nil {
		return nil, "", err
	}

	return &u, token, nil
}

// GetMe returns the console user by ID (from JWT).
func (s *Service) GetMe(ctx context.Context, userID string) (*ConsoleUser, error) {
	var u ConsoleUser
	var name sql.NullString
	err := s.db.QueryRowContext(ctx,
		"SELECT id, email, name, created_at, updated_at FROM console_users WHERE id = $1",
		userID).Scan(&u.ID, &u.Email, &name, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("console: user not found")
	}
	if err != nil {
		return nil, err
	}
	u.Name = name.String
	return &u, nil
}

// UpdateName updates a console user's name.
func (s *Service) UpdateName(ctx context.Context, userID, name string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE console_users SET name = $1 WHERE id = $2", name, userID)
	return err
}

// UpdateEmail updates a console user's email.
func (s *Service) UpdateEmail(ctx context.Context, userID, email string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE console_users SET email = $1 WHERE id = $2", email, userID)
	return err
}

// UpdatePassword updates a console user's password after verifying the old one.
func (s *Service) UpdatePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	var hash string
	err := s.db.QueryRowContext(ctx, "SELECT password_hash FROM console_users WHERE id = $1", userID).Scan(&hash)
	if err != nil {
		return fmt.Errorf("console: user not found")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(oldPassword)); err != nil {
		return fmt.Errorf("console: invalid current password")
	}
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "UPDATE console_users SET password_hash = $1 WHERE id = $2", string(newHash), userID)
	return err
}

// DeleteUser removes a console user.
func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM console_users WHERE id = $1", userID)
	return err
}

// CountUsers returns the total number of console users.
func (s *Service) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM console_users").Scan(&count)
	return count, err
}

// SignupEnabled returns whether signup is allowed.
// If CONSOLE_SIGNUP_ENABLED is "auto" (default), signup is allowed only when no users exist.
// If "true", signup is always allowed. If "false", signup is never allowed.
func (s *Service) SignupEnabled(ctx context.Context, setting string) (bool, error) {
	switch setting {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default: // "auto"
		count, err := s.CountUsers(ctx)
		if err != nil {
			return false, err
		}
		return count == 0, nil
	}
}

// FirstRun reports whether this instance has no console users yet, meaning the
// next account created owns it.
func (s *Service) FirstRun(ctx context.Context) (bool, error) {
	count, err := s.CountUsers(ctx)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// Invite describes a pending organization invite, identified by its token.
type Invite struct {
	Email            string `json:"email"`
	Name             string `json:"name"`
	Role             string `json:"role"`
	OrganizationID   string `json:"organizationId"`
	OrganizationName string `json:"organizationName"`
	// HasAccount reports whether this address already has a console account,
	// in which case the invite is accepted by signing in rather than by
	// registering.
	HasAccount bool `json:"hasAccount"`
}

// LookupInvite resolves a pending invite by its token. The token is the
// credential: nothing about an invite is discoverable without it.
func (s *Service) LookupInvite(ctx context.Context, token string) (*Invite, error) {
	var inv Invite
	var name, orgName sql.NullString
	err := s.db.QueryRowContext(ctx,
		`SELECT m.email, m.name, m.role, m.org_id, o.name
		   FROM organization_members m
		   JOIN organizations o ON o.id = m.org_id
		  WHERE m.invite_token = $1 AND m.status = 'pending'`,
		token).Scan(&inv.Email, &name, &inv.Role, &inv.OrganizationID, &orgName)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("console: invite not found or already used")
	}
	if err != nil {
		return nil, fmt.Errorf("console: lookup invite: %w", err)
	}
	inv.Name, inv.OrganizationName = name.String, orgName.String

	var n int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM console_users WHERE lower(email) = lower($1)", inv.Email).Scan(&n); err != nil {
		return nil, fmt.Errorf("console: lookup invite: %w", err)
	}
	inv.HasAccount = n > 0
	return &inv, nil
}

// RedeemInvite creates the account an invite was issued for and activates the
// membership in one transaction.
//
// This is deliberately not signup. The address is read from the invite rather
// than supplied by the caller, so possession of the token — not knowledge of
// who was invited — is what grants the account. And because the membership is
// activated here, the person lands inside the organization instead of holding
// an account that still has to accept something.
func (s *Service) RedeemInvite(ctx context.Context, token, password, name string) (*ConsoleUser, string, error) {
	inv, err := s.LookupInvite(ctx, token)
	if err != nil {
		return nil, "", err
	}
	if inv.HasAccount {
		return nil, "", fmt.Errorf("console: an account already exists for %s — sign in to accept", inv.Email)
	}
	if name == "" {
		name = inv.Name
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, "", fmt.Errorf("console: hash password: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", fmt.Errorf("console: redeem invite: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	id := uid.New("unique()")
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO console_users (id, email, name, password_hash, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, inv.Email, name, string(hash), now, now); err != nil {
		return nil, "", fmt.Errorf("console: redeem invite: %w", err)
	}

	// Consumes the token, so an invite cannot be redeemed twice.
	res, err := tx.ExecContext(ctx,
		`UPDATE organization_members
		    SET user_id = $1, name = $2, status = 'active', invite_token = NULL
		  WHERE invite_token = $3 AND status = 'pending'`,
		id, name, token)
	if err != nil {
		return nil, "", fmt.Errorf("console: redeem invite: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, "", fmt.Errorf("console: invite not found or already used")
	}

	if err := tx.Commit(); err != nil {
		return nil, "", fmt.Errorf("console: redeem invite: %w", err)
	}

	jwt, err := s.signJWT(id, inv.Email, name)
	if err != nil {
		return nil, "", err
	}
	return &ConsoleUser{ID: id, Email: inv.Email, Name: name, CreatedAt: now, UpdatedAt: now}, jwt, nil
}

// LoginOrCreateByOAuth finds an existing console user by email or creates one
// if signup is currently enabled. OAuth users have no password.
func (s *Service) LoginOrCreateByOAuth(ctx context.Context, email, name, provider, signupSetting string) (*ConsoleUser, string, error) {
	var u ConsoleUser
	var nameNull sql.NullString

	err := s.db.QueryRowContext(ctx,
		"SELECT id, email, name, created_at, updated_at FROM console_users WHERE email = $1",
		email).Scan(&u.ID, &u.Email, &nameNull, &u.CreatedAt, &u.UpdatedAt)

	if err == sql.ErrNoRows {
		// First time — only allowed when signup is enabled.
		enabled, err2 := s.SignupEnabled(ctx, signupSetting)
		if err2 != nil {
			return nil, "", err2
		}
		if !enabled {
			return nil, "", fmt.Errorf("console: signup disabled")
		}
		id := uid.New("unique()")
		now := time.Now().UTC()
		if name == "" {
			name = email
		}
		_, err2 = s.db.ExecContext(ctx,
			`INSERT INTO console_users (id, email, name, password_hash, created_at, updated_at)
			 VALUES ($1, $2, $3, '', $4, $5)`,
			id, email, name, now, now)
		if err2 != nil {
			return nil, "", fmt.Errorf("console: create oauth user: %w", err2)
		}
		u = ConsoleUser{ID: id, Email: email, Name: name, CreatedAt: now, UpdatedAt: now}
	} else if err != nil {
		return nil, "", err
	} else {
		u.Name = nameNull.String
	}

	token, err := s.signJWT(u.ID, u.Email, "")
	if err != nil {
		return nil, "", err
	}
	return &u, token, nil
}

// RequestPasswordReset generates a 1-hour reset token for the given email.
// Returns (token, true, nil) when the email is found, or ("", false, nil) when not.
// The caller decides whether to email the token or surface it directly.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) (token string, found bool, err error) {
	var userID string
	err = s.db.QueryRowContext(ctx,
		"SELECT id FROM console_users WHERE email = $1", email).Scan(&userID)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	b := make([]byte, 24)
	if _, err = rand.Read(b); err != nil {
		return "", false, fmt.Errorf("console: generate reset token: %w", err)
	}
	token = hex.EncodeToString(b)
	resetTokens.Store(token, resetEntry{userID: userID, expiresAt: time.Now().Add(time.Hour)})
	return token, true, nil
}

// ConfirmPasswordReset validates the token and sets a new password.
func (s *Service) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	val, ok := resetTokens.Load(token)
	if !ok {
		return fmt.Errorf("invalid or expired reset token")
	}
	entry := val.(resetEntry)
	if time.Now().After(entry.expiresAt) {
		resetTokens.Delete(token)
		return fmt.Errorf("invalid or expired reset token")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		"UPDATE console_users SET password_hash = $1, updated_at = $2 WHERE id = $3",
		string(hash), time.Now().UTC(), entry.userID)
	if err == nil {
		resetTokens.Delete(token)
	}
	return err
}

// ValidateToken parses and validates a console JWT, returning the user ID.
func (s *Service) ValidateToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &ConsoleClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return "", fmt.Errorf("console: invalid token")
	}
	claims, ok := token.Claims.(*ConsoleClaims)
	if !ok || !token.Valid || !claims.Console {
		return "", fmt.Errorf("console: invalid token")
	}
	return claims.Subject, nil
}

func (s *Service) signJWT(userID, email, sessionID string) (string, error) {
	claims := ConsoleClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Console:   true,
		SessionID: sessionID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
