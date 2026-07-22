package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/mittolabs/applad/internal/apperr"
)

// ConsoleSessionValidator is implemented by console.Service. Validation goes
// through the session table, not just the signature, so a revoked session is
// refused here too.
type ConsoleSessionValidator interface {
	ValidateSession(ctx context.Context, token string) (userID, sessionID string, err error)
}

// ConsoleAccessChecker is implemented by console.Service. It answers the only
// question a console credential leaves open: is this administrator allowed to
// touch this project's org?
type ConsoleAccessChecker interface {
	CanAccessProject(ctx context.Context, userID, projectID string) (bool, error)
}

// ConsoleUserFromContext returns the console user id stored by
// RequireConsoleAuth or by Authenticate for console JWTs.
func ConsoleUserFromContext(ctx context.Context) string {
	v, _ := ctx.Value(consoleUserKey).(string)
	return v
}

// RequireConsoleAuth guards console-level management routes (/projects,
// /organizations). These carry no project header, so the console JWT is the
// only identity — a request without a valid one gets nothing, which is what
// stops an anonymous curl from listing every project.
func RequireConsoleAuth(v ConsoleSessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := ""
			if auth := r.Header.Get("Authorization"); auth != "" {
				parts := strings.SplitN(auth, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
					tok = parts[1]
				}
			}
			// Fallback: the HttpOnly cookie the console login sets.
			if tok == "" {
				if c, err := r.Cookie("a_session_console"); err == nil {
					tok = c.Value
				}
			}
			if tok == "" {
				apperr.Unauthorized(w)
				return
			}
			userID, sessionID, err := v.ValidateSession(r.Context(), tok)
			if err != nil {
				apperr.Unauthorized(w)
				return
			}
			ctx := context.WithValue(r.Context(), consoleAdminKey, true)
			ctx = context.WithValue(ctx, consoleUserKey, userID)
			if sessionID != "" {
				ctx = context.WithValue(ctx, sessionKey, sessionID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
