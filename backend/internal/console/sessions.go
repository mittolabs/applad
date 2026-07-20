package console

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mittolabs/applad/internal/uid"
)

// Session is a tracked console sign-in.
type Session struct {
	ID         string    `json:"$id"`
	UserAgent  string    `json:"userAgent"`
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"$createdAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	Current    bool      `json:"current"`
}

// CreateSessionToken records a new session for the user and returns a JWT that
// carries its id, so the session can later be listed and revoked.
func (s *Service) CreateSessionToken(ctx context.Context, userID, email, userAgent, ip string) (string, error) {
	sessionID := uid.New("unique()")
	now := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO console_sessions (id, user_id, user_agent, ip, created_at, last_seen_at)
		 VALUES ($1, $2, $3, $4, $5, $5)`,
		sessionID, userID, userAgent, ip, now,
	); err != nil {
		return "", fmt.Errorf("console: create session: %w", err)
	}
	return s.signJWT(userID, email, sessionID)
}

// ValidateSession verifies a console JWT and, when it carries a session id,
// confirms the session is still active (not revoked). Returns the user id and
// session id. Tokens issued before sessions existed (no sid) stay valid so
// existing logins keep working; they simply have no session record.
func (s *Service) ValidateSession(ctx context.Context, tokenStr string) (userID, sessionID string, err error) {
	token, perr := jwt.ParseWithClaims(tokenStr, &ConsoleClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})
	if perr != nil {
		return "", "", fmt.Errorf("console: invalid token")
	}
	claims, ok := token.Claims.(*ConsoleClaims)
	if !ok || !token.Valid || !claims.Console {
		return "", "", fmt.Errorf("console: invalid token")
	}

	if claims.SessionID != "" {
		var revoked sql.NullTime
		qerr := s.db.QueryRowContext(ctx,
			"SELECT revoked_at FROM console_sessions WHERE id = $1 AND user_id = $2",
			claims.SessionID, claims.Subject,
		).Scan(&revoked)
		if qerr == sql.ErrNoRows || (qerr == nil && revoked.Valid) {
			return "", "", fmt.Errorf("console: session revoked")
		}
		if qerr != nil {
			return "", "", fmt.Errorf("console: validate session: %w", qerr)
		}
		// Best-effort activity timestamp; ignore errors.
		_, _ = s.db.ExecContext(ctx,
			"UPDATE console_sessions SET last_seen_at = NOW() WHERE id = $1", claims.SessionID)
	}

	return claims.Subject, claims.SessionID, nil
}

// ListSessions returns a user's active (non-revoked) sessions, newest first,
// flagging the one matching currentSessionID.
func (s *Service) ListSessions(ctx context.Context, userID, currentSessionID string) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_agent, ip, created_at, last_seen_at
		   FROM console_sessions
		  WHERE user_id = $1 AND revoked_at IS NULL
		  ORDER BY last_seen_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Session{}
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.UserAgent, &sess.IP, &sess.CreatedAt, &sess.LastSeenAt); err != nil {
			continue
		}
		sess.Current = sess.ID == currentSessionID && currentSessionID != ""
		out = append(out, sess)
	}
	return out, nil
}

// RevokeSession marks one of the user's sessions revoked (ownership enforced).
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID string) error {
	res, err := s.db.ExecContext(ctx,
		"UPDATE console_sessions SET revoked_at = NOW() WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL",
		sessionID, userID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("console: session not found")
	}
	return nil
}
