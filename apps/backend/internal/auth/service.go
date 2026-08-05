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
	mailer    EmailSender // optional; set for sessionAlerts emails
}

// NewService creates a new auth Service.
func NewService(database *db.DB, jwtSecret string) *Service {
	return &Service{db: database, jwtSecret: jwtSecret}
}

// SetMailer wires an email sender used by the sessionAlerts policy. It is
// optional: when nil, session-alert emails are simply not sent.
func (s *Service) SetMailer(m EmailSender) { s.mailer = m }

// Password-policy errors. The code before the colon is stable so handlers and
// SDKs can map it; the text after is human-readable.
var (
	errPasswordPersonalData = fmt.Errorf("password_personal_data: password must not contain your name or email address")
	errPasswordInDictionary = fmt.Errorf("password_in_dictionary: password is too common; choose a less predictable one")
	errPasswordReused       = fmt.Errorf("password_reused: password matches one you have used recently; choose a new one")
)

// authSecurity mirrors projects.AuthSecurity for enforcement without a cross-package import.
type authSecurity struct {
	UsersLimit                 int  `json:"usersLimit"`
	SessionLengthSeconds       int  `json:"sessionLengthSeconds"`
	SessionsPerUser            int  `json:"sessionsPerUser"`
	PasswordMinLength          int  `json:"passwordMinLength"`
	PasswordHistory            int  `json:"passwordHistory"`
	PasswordDictionary         bool `json:"passwordDictionary"`
	PasswordPersonalData       bool `json:"passwordPersonalData"`
	MFARequired                bool `json:"mfaRequired"`
	SessionAlerts              bool `json:"sessionAlerts"`
	InvalidateOnPasswordChange bool `json:"invalidateOnPasswordChange"`
}

func (s *Service) loadSecurity(ctx context.Context, projectID string) authSecurity {
	sec := authSecurity{
		SessionLengthSeconds:       365 * 24 * 3600,
		SessionsPerUser:            10,
		PasswordMinLength:          8,
		InvalidateOnPasswordChange: true,
	}
	var raw sql.NullString
	if err := s.db.QueryRowContext(ctx,
		"SELECT auth_config FROM projects WHERE id = $1", projectID).Scan(&raw); err != nil {
		return sec
	}
	if !raw.Valid || raw.String == "" || raw.String == "null" {
		return sec
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw.String), &cfg); err != nil {
		return sec
	}
	if secRaw, ok := cfg["security"]; ok {
		_ = json.Unmarshal(secRaw, &sec)
	}
	return sec
}

// CreateAccount creates a new user account.
func (s *Service) CreateAccount(ctx context.Context, projectID, userID, email, password, name string) (*model.User, error) {
	sec := s.loadSecurity(ctx, projectID)

	// Enforce password minimum length.
	if sec.PasswordMinLength > 0 && len(password) < sec.PasswordMinLength {
		return nil, fmt.Errorf("password_too_short: password must be at least %d characters", sec.PasswordMinLength)
	}

	// Enforce value-based password policy (personal data, dictionary).
	if err := s.checkPasswordPolicy(sec, password, email, name); err != nil {
		return nil, err
	}

	// Enforce users limit.
	if sec.UsersLimit > 0 {
		var count int
		if err := s.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM users WHERE project_id = $1", projectID).Scan(&count); err == nil {
			if count >= sec.UsersLimit {
				return nil, fmt.Errorf("users_limit_reached: project has reached its user limit of %d", sec.UsersLimit)
			}
		}
	}

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
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, projectID, email, name, string(hash), labelsJSON, prefsJSON, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "duplicate") {
			return nil, fmt.Errorf("user_already_exists: email already in use")
		}
		return nil, fmt.Errorf("auth: create account: %w", err)
	}
	if sec.PasswordHistory > 0 {
		s.recordPasswordHistory(ctx, projectID, id, string(hash))
	}
	s.logUserEvent(ctx, projectID, id, "user.create", "", "")
	return s.GetAccount(ctx, id, projectID)
}

// ImportedUser is a user account carried in from another platform by the
// migration engine, with a password credential already produced elsewhere.
type ImportedUser struct {
	ID             string // optional; generated when empty
	Email          string
	Phone          string
	Name           string
	PasswordHash   string         // may be empty for OAuth-only accounts
	PasswordAlgo   string         // one of the auth.Algo* values; defaults to bcrypt
	PasswordParams map[string]any // algorithm-specific verify parameters, may be nil
	EmailVerified  bool
	PhoneVerified  bool
	Labels         []string
	Prefs          map[string]any
	CreatedAt      time.Time // optional; defaults to now
}

// ImportUser inserts a user with a pre-computed password credential (hash +
// algorithm + params) that another platform produced, bypassing Applad's own
// bcrypt hashing. The account verifies against the foreign algorithm at first
// sign-in and is transparently re-hashed to bcrypt then (see checkAndRehash).
//
// It is idempotent on (project_id, email): importing the same account twice
// returns the existing user's ID rather than failing, so a migration can resume.
// Returns the resolved user ID.
func (s *Service) ImportUser(ctx context.Context, projectID string, u ImportedUser) (string, error) {
	id := u.ID
	if id == "" {
		id = uid.New("")
	}
	algo := u.PasswordAlgo
	if algo == "" {
		algo = AlgoBcrypt
	}
	now := time.Now().UTC()
	created := u.CreatedAt
	if created.IsZero() {
		created = now
	}
	labels := u.Labels
	if labels == nil {
		labels = []string{}
	}
	labelsJSON, _ := json.Marshal(labels)
	prefs := u.Prefs
	if prefs == nil {
		prefs = map[string]interface{}{}
	}
	prefsJSON, _ := json.Marshal(prefs)

	var hash sql.NullString
	if u.PasswordHash != "" {
		hash = sql.NullString{String: u.PasswordHash, Valid: true}
	}
	var email sql.NullString
	if u.Email != "" {
		email = sql.NullString{String: u.Email, Valid: true}
	}
	var phone sql.NullString
	if u.Phone != "" {
		phone = sql.NullString{String: u.Phone, Valid: true}
	}
	var paramsJSON []byte
	if len(u.PasswordParams) > 0 {
		paramsJSON, _ = json.Marshal(u.PasswordParams)
	}

	// Idempotency is keyed on the primary key (id): the migration engine passes a
	// stable, source-derived id, so re-running or resuming a job updates the same
	// row rather than inserting a duplicate. This is why email cannot be the
	// conflict key: it is NULL for OAuth/phone accounts and NULLs never conflict,
	// so an email-based upsert would duplicate every email-less user on each run.
	// A different account already holding this email trips the (project_id,email)
	// unique constraint and returns an error the caller treats as "exists".
	var resolvedID string
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO users
		   (id, project_id, email, phone, name, password_hash, password_algo, password_params,
		    email_verified, phone_verified, status, labels, prefs, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,1,$11,$12,$13,$13)
		 ON CONFLICT (id) DO UPDATE SET updated_at = EXCLUDED.updated_at
		 RETURNING id`,
		id, projectID, email, phone, u.Name, hash, algo, paramsJSON,
		u.EmailVerified, u.PhoneVerified, labelsJSON, prefsJSON, created).Scan(&resolvedID)
	if err != nil {
		return "", fmt.Errorf("auth: import user: %w", err)
	}
	return resolvedID, nil
}

// GetAccount returns the user account by userID.
func (s *Service) GetAccount(ctx context.Context, userID, projectID string) (*model.User, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, email, phone, name, email_verified, phone_verified, status, labels, prefs, created_at, updated_at
		 FROM users WHERE id = $1 AND project_id = $2`, userID, projectID)
	return scanUser(row)
}

// GetUser is an alias for GetAccount (used by server-side user management).
func (s *Service) GetUser(ctx context.Context, userID, projectID string) (*model.User, error) {
	return s.GetAccount(ctx, userID, projectID)
}

// UpdateName updates a user's display name.
func (s *Service) UpdateName(ctx context.Context, userID, projectID, name string) (*model.User, error) {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET name = $1 WHERE id = $2 AND project_id = $3", name, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("auth: update name: %w", err)
	}
	return s.GetAccount(ctx, userID, projectID)
}

// UpdateEmail updates a user's email after verifying the password.
func (s *Service) UpdateEmail(ctx context.Context, userID, projectID, email, password string) (*model.User, error) {
	ok, err := s.verifyUserPassword(ctx, projectID, userID, password)
	if err != nil {
		return nil, fmt.Errorf("auth: user not found")
	}
	if !ok {
		return nil, fmt.Errorf("auth: invalid password")
	}
	_, err = s.db.ExecContext(ctx,
		"UPDATE users SET email = $1 WHERE id = $2 AND project_id = $3", email, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("auth: update email: %w", err)
	}
	return s.GetAccount(ctx, userID, projectID)
}

// UpdatePassword updates a user's password after verifying the old one.
func (s *Service) UpdatePassword(ctx context.Context, userID, projectID, password, oldPassword string) (*model.User, error) {
	sec := s.loadSecurity(ctx, projectID)

	if sec.PasswordMinLength > 0 && len(password) < sec.PasswordMinLength {
		return nil, fmt.Errorf("password_too_short: password must be at least %d characters", sec.PasswordMinLength)
	}

	var hash, algo string
	var paramsRaw []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT password_hash, password_algo, password_params FROM users WHERE id = $1 AND project_id = $2",
		userID, projectID).Scan(&hash, &algo, &paramsRaw)
	if err != nil {
		return nil, fmt.Errorf("auth: user not found")
	}
	if oldPassword != "" {
		var params map[string]any
		if len(paramsRaw) > 0 {
			_ = json.Unmarshal(paramsRaw, &params)
		}
		if ok, _ := verifyForeignPassword(algo, hash, params, oldPassword); !ok {
			return nil, fmt.Errorf("auth: invalid old password")
		}
	}

	// Personal-data enforcement needs the user's email and name; only pay for
	// the extra read when the policy is on.
	email, name := "", ""
	if sec.PasswordPersonalData {
		var e, n sql.NullString
		s.db.QueryRowContext(ctx,
			"SELECT email, name FROM users WHERE id = $1 AND project_id = $2", userID, projectID).Scan(&e, &n)
		email, name = e.String, n.String
	}
	if err := s.checkPasswordPolicy(sec, password, email, name); err != nil {
		return nil, err
	}

	// Reject reuse of the current or a recent password when history is on.
	if reused, err := s.passwordReused(ctx, projectID, userID, password, hash, sec.PasswordHistory); err == nil && reused {
		return nil, errPasswordReused
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx,
		"UPDATE users SET password_hash = $1, password_algo = 'bcrypt', password_params = NULL WHERE id = $2 AND project_id = $3",
		string(newHash), userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("auth: update password: %w", err)
	}
	if sec.PasswordHistory > 0 {
		s.recordPasswordHistory(ctx, projectID, userID, string(newHash))
	}
	// Invalidate all sessions when password changes if configured.
	if sec.InvalidateOnPasswordChange {
		s.db.ExecContext(ctx,
			"DELETE FROM sessions WHERE user_id = $1 AND project_id = $2", userID, projectID)
	}
	s.logUserEvent(ctx, projectID, userID, "user.password_change", "", "")
	return s.GetAccount(ctx, userID, projectID)
}

// UpdatePrefs updates a user's preferences.
func (s *Service) UpdatePrefs(ctx context.Context, userID, projectID string, prefs map[string]interface{}) (*model.User, error) {
	prefsJSON, err := json.Marshal(prefs)
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx,
		"UPDATE users SET prefs = $1 WHERE id = $2 AND project_id = $3", prefsJSON, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("auth: update prefs: %w", err)
	}
	return s.GetAccount(ctx, userID, projectID)
}

// DeleteAccount deletes a user account.
func (s *Service) DeleteAccount(ctx context.Context, userID, projectID string) error {
	s.logUserEvent(ctx, projectID, userID, "user.delete", "", "")
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM users WHERE id = $1 AND project_id = $2", userID, projectID)
	return err
}

// LoginResult carries the outcome of a password sign-in that may involve MFA.
// Exactly one of the terminal states holds: a Session is present (optionally with
// MFAEnrollmentRequired set), or MFAChallenge is true and no session was opened.
type LoginResult struct {
	Session *model.Session
	Token   string
	// MFAChallenge is true when the account has an enrolled factor and no valid
	// code was supplied. No session is issued; the caller must resupply a code.
	MFAChallenge bool
	// MFAEnrollmentRequired is true when the project requires MFA but this user
	// has no factor yet. A session IS issued (so the user can enrol and is not
	// locked out); the caller should route the user into MFA setup.
	MFAEnrollmentRequired bool
}

// errMFAInvalidCode is returned when an MFA-enrolled user supplies a bad code
// during sign-in. The code before the colon is stable for handlers and SDKs.
var errMFAInvalidCode = fmt.Errorf("user_mfa_invalid: invalid MFA code")

// userHasMFA reports whether the user has a verified, enabled MFA factor.
// A query error is treated as "no factor" so a transient read cannot become a
// lockout; the caller falls back to the ordinary password outcome.
func (s *Service) userHasMFA(ctx context.Context, projectID, userID string) bool {
	var enabled sql.NullBool
	if err := s.db.QueryRowContext(ctx,
		"SELECT mfa_enabled FROM users WHERE id = $1 AND project_id = $2",
		userID, projectID).Scan(&enabled); err != nil {
		return false
	}
	return enabled.Bool
}

// checkAndRehash verifies password against a credential produced by algo (with
// the given params, raw JSON from users.password_params) and, on a successful
// match with a non-native algorithm, transparently re-hashes to bcrypt. The
// rehash is best-effort: a failure to persist the upgrade never fails the check.
// Returns whether the password matched; an unsupported/malformed algorithm is a
// non-match with a returned error (callers that only care about the boolean can
// ignore it).
func (s *Service) checkAndRehash(ctx context.Context, projectID, userID, hash, algo string, paramsRaw []byte, password string) (bool, error) {
	var params map[string]any
	if len(paramsRaw) > 0 {
		_ = json.Unmarshal(paramsRaw, &params)
	}
	ok, err := verifyForeignPassword(algo, hash, params, password)
	if err != nil || !ok {
		return false, err
	}
	if passwordNeedsRehash(algo) {
		if nh, herr := bcrypt.GenerateFromPassword([]byte(password), 12); herr == nil {
			s.db.ExecContext(ctx,
				"UPDATE users SET password_hash = $1, password_algo = 'bcrypt', password_params = NULL WHERE id = $2 AND project_id = $3",
				string(nh), userID, projectID)
		}
	}
	return true, nil
}

// verifyUserPassword loads a user's stored credential and verifies password
// against it (rehashing to bcrypt on a successful foreign-algorithm match, via
// checkAndRehash). A missing user is (false, error).
func (s *Service) verifyUserPassword(ctx context.Context, projectID, userID, password string) (bool, error) {
	var hash, algo string
	var paramsRaw []byte
	if err := s.db.QueryRowContext(ctx,
		"SELECT password_hash, password_algo, password_params FROM users WHERE id = $1 AND project_id = $2",
		userID, projectID).Scan(&hash, &algo, &paramsRaw); err != nil {
		return false, err
	}
	return s.checkAndRehash(ctx, projectID, userID, hash, algo, paramsRaw, password)
}

// CreateEmailSession validates email+password and, when the account or project
// calls for it, enforces MFA before a session is opened:
//   - an account with an enrolled factor must present a valid TOTP or recovery
//     code; without one the result is an MFA challenge and no session;
//   - a project with mfaRequired=true but a user with no factor still gets a
//     session (never a lockout), flagged so the client can force enrolment.
func (s *Service) CreateEmailSession(ctx context.Context, projectID, email, password, code, ip, ua string) (*LoginResult, error) {
	var userID, hash, algo string
	var paramsRaw []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT id, password_hash, password_algo, password_params FROM users WHERE email = $1 AND project_id = $2 AND status = 1",
		email, projectID).Scan(&userID, &hash, &algo, &paramsRaw)
	if err != nil {
		return nil, fmt.Errorf("auth: invalid credentials")
	}
	if ok, _ := s.checkAndRehash(ctx, projectID, userID, hash, algo, paramsRaw, password); !ok {
		return nil, fmt.Errorf("auth: invalid credentials")
	}

	sec := s.loadSecurity(ctx, projectID)
	enrolled := s.userHasMFA(ctx, projectID, userID)

	if enrolled {
		// A user who enrolled MFA must complete the challenge on every sign-in,
		// independent of the project-wide mfaRequired setting.
		if code == "" {
			return &LoginResult{MFAChallenge: true}, nil
		}
		if err := s.ValidateMFAForLogin(ctx, projectID, email, code); err != nil {
			return nil, errMFAInvalidCode
		}
	}

	sess, token, err := s.createSession(ctx, userID, projectID, "email", ip, ua)
	if err != nil {
		return nil, err
	}
	res := &LoginResult{Session: sess, Token: token}
	if !enrolled && sec.MFARequired {
		res.MFAEnrollmentRequired = true
	}
	return res, nil
}

// CreateAnonymousSession creates an anonymous user and session.
// The anonymous user gets a generated email like anon_{uid}@anonymous.applad.local,
// no password, and status=1 (active). The provider is recorded as "anonymous".
func (s *Service) CreateAnonymousSession(ctx context.Context, projectID, ip, ua string) (*model.Session, string, error) {
	userID := uid.New("unique()")
	anonEmail := fmt.Sprintf("anon_%s@anonymous.applad.local", userID)
	now := time.Now().UTC()
	labelsJSON, _ := json.Marshal([]string{"anonymous"})
	prefsJSON, _ := json.Marshal(map[string]interface{}{})
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, project_id, email, name, labels, prefs, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, 1, $7, $8)`,
		userID, projectID, anonEmail, "Anonymous User", labelsJSON, prefsJSON, now, now)
	if err != nil {
		return nil, "", fmt.Errorf("auth: create anon user: %w", err)
	}
	return s.createSession(ctx, userID, projectID, "anonymous", ip, ua)
}

// SendPhoneOTP generates a 6-digit OTP, stores it in auth_tokens, returns the code.
func (s *Service) SendPhoneOTP(ctx context.Context, projectID, phone string) (string, error) {
	if phone == "" {
		return "", fmt.Errorf("auth: phone is required")
	}
	// Generate 6-digit OTP
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	code := fmt.Sprintf("%06d", n.Int64()+100000)

	tokenID := uid.New("unique()")
	expires := time.Now().UTC().Add(10 * time.Minute)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO auth_tokens (id, user_id, project_id, type, token, expires_at, created_at)
		 VALUES ($1, $2, $3, 'phone_otp', $4, $5, $6)`,
		tokenID, phone, projectID, code, expires, time.Now().UTC())
	if err != nil {
		return "", fmt.Errorf("auth: store phone OTP: %w", err)
	}
	return code, nil
}

// VerifyPhoneOTP verifies the OTP and creates a session. Creates user if needed.
func (s *Service) VerifyPhoneOTP(ctx context.Context, projectID, phone, code, ip, ua string) (*model.Session, string, error) {
	var tokenID string
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM auth_tokens WHERE user_id=$1 AND project_id=$2 AND type='phone_otp' AND token=$3 AND expires_at > $4`,
		phone, projectID, code, time.Now().UTC()).Scan(&tokenID)
	if err != nil {
		return nil, "", fmt.Errorf("auth: invalid or expired OTP")
	}
	// Delete used token
	s.db.ExecContext(ctx, "DELETE FROM auth_tokens WHERE id=$1", tokenID)

	// Find or create user by phone
	var userID string
	err = s.db.QueryRowContext(ctx,
		"SELECT id FROM users WHERE phone=$1 AND project_id=$2", phone, projectID).Scan(&userID)
	if err != nil {
		// Create new user
		userID = uid.New("unique()")
		now := time.Now().UTC()
		prefsJSON, _ := json.Marshal(map[string]interface{}{})
		_, err = s.db.ExecContext(ctx,
			`INSERT INTO users (id, project_id, phone, name, prefs, status, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, 1, $6, $7)`,
			userID, projectID, phone, "Phone User", prefsJSON, now, now)
		if err != nil {
			return nil, "", fmt.Errorf("auth: create phone user: %w", err)
		}
	}
	return s.createSession(ctx, userID, projectID, "phone", ip, ua)
}

// logUserEvent records an audit event in the user_logs table.
func (s *Service) logUserEvent(ctx context.Context, projectID, userID, event, ip, ua string) {
	s.db.ExecContext(ctx,
		`INSERT INTO user_logs (id, project_id, user_id, event, ip, user_agent, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uid.New("unique()"), projectID, userID, event, ip, ua, time.Now().UTC())
}

func (s *Service) createSession(ctx context.Context, userID, projectID, provider, ip, ua string) (*model.Session, string, error) {
	sec := s.loadSecurity(ctx, projectID)

	// Enforce sessions-per-user limit.
	if sec.SessionsPerUser > 0 {
		var count int
		if err := s.db.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM sessions WHERE user_id = $1 AND project_id = $2 AND expires_at > NOW()",
			userID, projectID).Scan(&count); err == nil && count >= sec.SessionsPerUser {
			// Evict the oldest session to stay within the limit.
			s.db.ExecContext(ctx,
				`DELETE FROM sessions WHERE id = (
				   SELECT id FROM sessions WHERE user_id = $1 AND project_id = $2
				   ORDER BY created_at ASC LIMIT 1
				 )`, userID, projectID)
		}
	}

	sessionID := uid.New("unique()")
	length := time.Duration(sec.SessionLengthSeconds) * time.Second
	if length <= 0 {
		length = 365 * 24 * time.Hour
	}
	expires := time.Now().UTC().Add(length)
	now := time.Now().UTC()

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO sessions (id, user_id, project_id, ip, user_agent, expires_at, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		sessionID, userID, projectID, ip, ua, expires, now)
	if err != nil {
		return nil, "", fmt.Errorf("auth: create session: %w", err)
	}

	token, err := s.signJWT(userID, sessionID, projectID, expires)
	if err != nil {
		return nil, "", err
	}

	// Record audit event
	s.logUserEvent(ctx, projectID, userID, "session.create."+provider, ip, ua)

	// Notify the user of a new password sign-in when the policy is on.
	if s.mailer != nil && sec.SessionAlerts {
		s.maybeSendSessionAlert(ctx, projectID, userID, provider, ip, ua)
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
		"SELECT id, user_id, ip, user_agent, expires_at, created_at FROM sessions WHERE id = $1 AND user_id = $2 AND project_id = $3",
		sessionID, userID, projectID)
	return scanSession(row)
}

// ListSessions returns all sessions for a user.
func (s *Service) ListSessions(ctx context.Context, userID, projectID string) ([]*model.Session, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, user_id, ip, user_agent, expires_at, created_at FROM sessions WHERE user_id = $1 AND project_id = $2 ORDER BY created_at DESC",
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
		"DELETE FROM sessions WHERE id = $1 AND user_id = $2 AND project_id = $3", sessionID, userID, projectID)
	return err
}

// DeleteSessions deletes all sessions for a user.
func (s *Service) DeleteSessions(ctx context.Context, userID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM sessions WHERE user_id = $1 AND project_id = $2", userID, projectID)
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
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
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
	n := 1
	var args []interface{}
	query := `SELECT id, project_id, email, phone, name, email_verified, phone_verified, status, labels, prefs, created_at, updated_at
	          FROM users WHERE project_id = $1`
	args = append(args, projectID)

	if search != "" {
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern)
		query += fmt.Sprintf(" AND (name LIKE $%d OR email LIKE $%d)", n+1, n+2)
		n += 2
	}
	args = append(args, limit, offset)
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", n+1, n+2)

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
	countQ := "SELECT COUNT(*) FROM users WHERE project_id = $1"
	countArgs := []interface{}{projectID}
	if search != "" {
		pattern := "%" + search + "%"
		countArgs = append(countArgs, pattern, pattern)
		countQ += fmt.Sprintf(" AND (name LIKE $%d OR email LIKE $%d)", len(countArgs)-1, len(countArgs))
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
		"UPDATE users SET status = $1 WHERE id = $2 AND project_id = $3", statusInt, userID, projectID)
	if err != nil {
		return nil, err
	}
	return s.GetAccount(ctx, userID, projectID)
}

// DeleteUser deletes a user.
func (s *Service) DeleteUser(ctx context.Context, userID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM users WHERE id = $1 AND project_id = $2", userID, projectID)
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
		"UPDATE users SET mfa_secret = $1, mfa_recovery = $2 WHERE id = $3 AND project_id = $4",
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
		"SELECT mfa_secret FROM users WHERE id = $1 AND project_id = $2",
		userID, projectID).Scan(&secretB32)
	if err != nil || secretB32 == "" {
		return fmt.Errorf("auth: MFA not set up")
	}

	if !validateTOTP(secretB32, code) {
		return fmt.Errorf("auth: invalid MFA code")
	}

	_, err = s.db.ExecContext(ctx,
		"UPDATE users SET mfa_enabled = 1 WHERE id = $1 AND project_id = $2",
		userID, projectID)
	return err
}

// DisableMFA disables MFA for a user.
func (s *Service) DisableMFA(ctx context.Context, userID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET mfa_enabled = 0, mfa_secret = NULL, mfa_recovery = NULL WHERE id = $1 AND project_id = $2",
		userID, projectID)
	return err
}

// CheckMFA returns whether MFA is enabled for the user with the given email.
func (s *Service) CheckMFA(ctx context.Context, projectID, email string) (bool, error) {
	var enabled bool
	err := s.db.QueryRowContext(ctx,
		"SELECT mfa_enabled FROM users WHERE email = $1 AND project_id = $2 AND status = 1",
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
		"SELECT mfa_secret, mfa_recovery FROM users WHERE email = $1 AND project_id = $2",
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
				"UPDATE users SET mfa_recovery = $1 WHERE email = $2 AND project_id = $3",
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
		"INSERT INTO auth_tokens (id, user_id, project_id, type, secret, expires_at) VALUES ($1, $2, $3, $4, $5, $6)",
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
		"SELECT id, user_id, expires_at FROM auth_tokens WHERE project_id = $1 AND type = $2 AND secret = $3",
		projectID, tokenType, secret).Scan(&id, &userID, &expiresAt)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("auth: invalid or expired token")
	}
	if err != nil {
		return "", err
	}
	if time.Now().After(expiresAt) {
		s.db.ExecContext(ctx, "DELETE FROM auth_tokens WHERE id = $1", id)
		return "", fmt.Errorf("auth: token expired")
	}
	// Consume token
	s.db.ExecContext(ctx, "DELETE FROM auth_tokens WHERE id = $1", id)
	return userID, nil
}

// CreateMagicLinkToken creates a magic link token for passwordless login.
func (s *Service) CreateMagicLinkToken(ctx context.Context, projectID, email string) (string, error) {
	var userID string
	err := s.db.QueryRowContext(ctx,
		"SELECT id FROM users WHERE email = $1 AND project_id = $2",
		email, projectID).Scan(&userID)
	if err == sql.ErrNoRows {
		// Create user without password
		userID = uid.New("unique()")
		now := time.Now().UTC()
		labelsJSON, _ := json.Marshal([]string{})
		prefsJSON, _ := json.Marshal(map[string]interface{}{})
		_, err = s.db.ExecContext(ctx,
			"INSERT INTO users (id, project_id, email, labels, prefs, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
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
	s.db.ExecContext(ctx, "UPDATE users SET email_verified = 1 WHERE id = $1 AND project_id = $2", userID, projectID)
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
		"UPDATE users SET email_verified = 1 WHERE id = $1 AND project_id = $2", userID, projectID)
	return err
}

// CreatePasswordResetToken creates a password reset token.
func (s *Service) CreatePasswordResetToken(ctx context.Context, projectID, email string) (string, error) {
	var userID string
	err := s.db.QueryRowContext(ctx,
		"SELECT id FROM users WHERE email = $1 AND project_id = $2",
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
	sec := s.loadSecurity(ctx, projectID)
	if sec.PasswordMinLength > 0 && len(newPassword) < sec.PasswordMinLength {
		return fmt.Errorf("password_too_short: password must be at least %d characters", sec.PasswordMinLength)
	}

	// Load the current credentials only for the policies that need them.
	var curHash, email, name sql.NullString
	if sec.PasswordPersonalData || sec.PasswordHistory > 0 {
		s.db.QueryRowContext(ctx,
			"SELECT password_hash, email, name FROM users WHERE id = $1 AND project_id = $2",
			userID, projectID).Scan(&curHash, &email, &name)
	}
	if err := s.checkPasswordPolicy(sec, newPassword, email.String, name.String); err != nil {
		return err
	}
	if reused, err := s.passwordReused(ctx, projectID, userID, newPassword, curHash.String, sec.PasswordHistory); err == nil && reused {
		return errPasswordReused
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		"UPDATE users SET password_hash = $1, password_algo = 'bcrypt', password_params = NULL WHERE id = $2 AND project_id = $3",
		string(hash), userID, projectID)
	if err != nil {
		return err
	}
	if sec.PasswordHistory > 0 {
		s.recordPasswordHistory(ctx, projectID, userID, string(hash))
	}
	return nil
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
func (s *Service) CreateOAuthSession(ctx context.Context, projectID, provider, oauthID, email, name string, emailVerified bool, ip, ua string) (*model.Session, string, error) {
	// 1. An exact identity match (provider + oauth_id) always wins.
	var userID string
	err := s.db.QueryRowContext(ctx,
		"SELECT id FROM users WHERE project_id = $1 AND oauth_provider = $2 AND oauth_id = $3",
		projectID, provider, oauthID).Scan(&userID)
	if err == nil {
		return s.createSession(ctx, userID, projectID, provider, ip, ua)
	}
	if err != sql.ErrNoRows {
		return nil, "", err
	}

	// 2. No identity match. Only a VERIFIED, non-empty email may attach this
	// OAuth identity to an existing account. An unverified or provider-chosen
	// email must not, or it becomes an account-takeover vector; and an empty
	// email (emailless providers) is never a match key, or the first emailless
	// user's account would swallow every later one.
	if email != "" {
		var existingID string
		e := s.db.QueryRowContext(ctx,
			"SELECT id FROM users WHERE project_id = $1 AND email = $2",
			projectID, email).Scan(&existingID)
		if e == nil {
			if !emailVerified {
				return nil, "", fmt.Errorf("oauth_email_unverified: an account with this email already exists; sign in with it and link %s from your account", provider)
			}
			if _, uerr := s.db.ExecContext(ctx,
				"UPDATE users SET oauth_provider = $1, oauth_id = $2, email_verified = 1 WHERE id = $3 AND project_id = $4",
				provider, oauthID, existingID, projectID); uerr != nil {
				return nil, "", uerr
			}
			return s.createSession(ctx, existingID, projectID, provider, ip, ua)
		}
		if e != sql.ErrNoRows {
			return nil, "", e
		}
	}

	// 3. Create a new account. Store the email only if present, and mark it
	// verified only if the provider asserted verification.
	userID = uid.New("unique()")
	now := time.Now().UTC()
	labelsJSON, _ := json.Marshal([]string{})
	prefsJSON, _ := json.Marshal(map[string]interface{}{})
	var emailArg interface{}
	if email != "" {
		emailArg = email
	}
	verified := 0
	if emailVerified {
		verified = 1
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, project_id, email, name, oauth_provider, oauth_id, email_verified, labels, prefs, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		userID, projectID, emailArg, name, provider, oauthID, verified, labelsJSON, prefsJSON, now, now); err != nil {
		return nil, "", fmt.Errorf("auth: create oauth user: %w", err)
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
		"UPDATE users SET email = $1 WHERE id = $2 AND project_id = $3", email, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("auth: update email admin: %w", err)
	}
	return s.GetAccount(ctx, userID, projectID)
}
