package auth

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/uid"
	"golang.org/x/crypto/bcrypt"
)

// personalDataMinToken is the shortest personal-data token the password check
// considers. Very short fragments (a two-letter name, the "io" in a domain)
// would reject almost any password, so tokens below this length are ignored.
const personalDataMinToken = 3

// passwordContainsPersonalData reports whether the password contains the user's
// email local-part or a word from their name, compared case-insensitively. It
// backs the passwordPersonalData policy: a password that embeds who you are is
// rejected with password_personal_data.
func passwordContainsPersonalData(password, email, name string) bool {
	pw := strings.ToLower(strings.TrimSpace(password))
	if pw == "" {
		return false
	}
	var tokens []string
	if email != "" {
		local := email
		if at := strings.IndexByte(email, '@'); at > 0 {
			local = email[:at]
		}
		tokens = append(tokens, local)
		// Split the local-part on common separators so "grace.hopper" also
		// contributes "grace" and "hopper".
		tokens = append(tokens, strings.FieldsFunc(local, func(r rune) bool {
			return r == '.' || r == '_' || r == '-' || r == '+'
		})...)
	}
	tokens = append(tokens, strings.Fields(name)...)
	for _, tok := range tokens {
		tok = strings.ToLower(strings.TrimSpace(tok))
		if len(tok) < personalDataMinToken {
			continue
		}
		if strings.Contains(pw, tok) {
			return true
		}
	}
	return false
}

// checkPasswordPolicy applies the value-based password rules (personal data and
// dictionary) that do not need extra database access. minLength is checked by
// the caller because its error message includes the configured length.
// email and name may be empty when the caller has not loaded them (personal-data
// enforcement is skipped in that case).
func (s *Service) checkPasswordPolicy(sec authSecurity, password, email, name string) error {
	if sec.PasswordPersonalData && passwordContainsPersonalData(password, email, name) {
		return errPasswordPersonalData
	}
	if sec.PasswordDictionary && isCommonPassword(password) {
		return errPasswordInDictionary
	}
	return nil
}

// recordPasswordHistory appends a bcrypt hash to the user's password history.
// It is best-effort: a failure to record must not fail the password change that
// already succeeded, mirroring how audit events are logged.
func (s *Service) recordPasswordHistory(ctx context.Context, projectID, userID, hash string) {
	s.db.ExecContext(ctx,
		`INSERT INTO password_history (id, user_id, project_id, password_hash, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		uid.New("unique()"), userID, projectID, hash, time.Now().UTC())
}

// passwordReused reports whether the candidate matches any of the user's last N
// stored password hashes, or their current hash. currentHash may be empty. It
// backs the passwordHistory policy and returns password_reused when N > 0 and a
// match is found.
func (s *Service) passwordReused(ctx context.Context, projectID, userID, candidate, currentHash string, n int) (bool, error) {
	if n <= 0 {
		return false, nil
	}
	if currentHash != "" && bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(candidate)) == nil {
		return true, nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT password_hash FROM password_history
		 WHERE user_id = $1 AND project_id = $2
		 ORDER BY created_at DESC LIMIT $3`,
		userID, projectID, n)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return false, err
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(candidate)) == nil {
			return true, nil
		}
	}
	return false, nil
}

// shouldSendSessionAlert decides whether a new session warrants a "new sign-in"
// email under the sessionAlerts policy. It is a pure function so the skip
// conditions are unit-testable without a mailer or a database.
//
// Alerts are sent only for password sign-ins ("email"). They are skipped for:
//   - magic-link, email/phone OTP, OAuth2 and anonymous sessions (the provider
//     is not "email"), which already prove control of an inbox or account;
//   - the very first session for a user (nothing to compare against, and it is
//     almost always the sign-in right after signup);
//   - users with no email on file, or a generated anonymous address.
//
// totalSessions is the number of sessions the user has including the one just
// created, so the first session is totalSessions == 1.
func shouldSendSessionAlert(enabled bool, provider, email string, totalSessions int) bool {
	if !enabled {
		return false
	}
	if provider != "email" {
		return false
	}
	if email == "" || strings.HasSuffix(email, "@anonymous.applad.local") {
		return false
	}
	if totalSessions <= 1 {
		return false
	}
	return true
}

// maybeSendSessionAlert sends a "new sign-in" email when the sessionAlerts
// policy applies to the just-created session. It is best-effort: any failure to
// read the user or send the mail is swallowed so a session is never blocked on
// an alert. The caller has already checked that a mailer is configured and the
// policy is on.
func (s *Service) maybeSendSessionAlert(ctx context.Context, projectID, userID, provider, ip, ua string) {
	if provider != "email" {
		return
	}
	var total int
	s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sessions WHERE user_id = $1 AND project_id = $2",
		userID, projectID).Scan(&total)
	var email sql.NullString
	s.db.QueryRowContext(ctx,
		"SELECT email FROM users WHERE id = $1 AND project_id = $2",
		userID, projectID).Scan(&email)
	if !shouldSendSessionAlert(true, provider, email.String, total) {
		return
	}
	subject := "New sign-in to your account"
	body := fmt.Sprintf(
		"<p>A new sign-in to your account was just recorded.</p>"+
			"<p>IP address: %s<br>Device: %s</p>"+
			"<p>If this was you, no action is needed. If it was not, change your password now.</p>",
		html.EscapeString(ip), html.EscapeString(ua))
	_ = s.mailer.SendEmail(ctx, []string{email.String}, subject, body)
}
