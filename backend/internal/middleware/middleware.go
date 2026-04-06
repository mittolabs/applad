package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mittolabs/applad/internal/apperr"
)

// contextKey is a typed key to avoid collisions.
type contextKey int

const (
	projectKey contextKey = iota
	userKey
	sessionKey
	apiKeyKey
	projectIDKey
)

// Claims is the JWT claims structure.
type Claims struct {
	jwt.RegisteredClaims
	SessionID string `json:"sid"`
	ProjectID string `json:"pid"`
}

// ProjectProvider is implemented by the projects service.
type ProjectProvider interface {
	GetByKey(ctx context.Context, secret string) (interface{}, error)
	Get(ctx context.Context, id string) (interface{}, error)
}

// WriteJSON writes v as JSON to w.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// WriteError writes an error JSON response.
func WriteError(w http.ResponseWriter, status int, errType, message string) {
	apperr.Write(w, status, errType, message)
}

// ProjectFromContext returns the project ID stored by ProjectContext middleware.
func ProjectFromContext(ctx context.Context) string {
	v, _ := ctx.Value(projectIDKey).(string)
	return v
}

// UserFromContext returns the userID stored by Authenticate middleware.
func UserFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userKey).(string)
	return v
}

// SessionFromContext returns the session ID stored by Authenticate middleware.
func SessionFromContext(ctx context.Context) string {
	v, _ := ctx.Value(sessionKey).(string)
	return v
}

// IsAPIKey returns true if the request was authenticated via API key.
func IsAPIKey(ctx context.Context) bool {
	v, _ := ctx.Value(apiKeyKey).(bool)
	return v
}

// CORS applies permissive CORS headers.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Applad-Project, X-Applad-Key, X-Applad-Response-Format")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ProjectContext reads the x-applad-project header and stores the project ID in context.
func ProjectContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projectID := r.Header.Get("X-Applad-Project")
		if projectID == "" {
			apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "Missing X-Applad-Project header")
			return
		}
		ctx := context.WithValue(r.Context(), projectIDKey, projectID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Authenticate reads either x-applad-key or Authorization: Bearer <jwt> and validates.
// It does NOT reject unauthenticated requests; use RequireAuth for that.
func Authenticate(jwtSecret string, _ interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Try API key first
			if apiKey := r.Header.Get("X-Applad-Key"); apiKey != "" {
				// Validate format: applad_key_<hex>
				if strings.HasPrefix(apiKey, "applad_key_") {
					hash := fmt.Sprintf("%x", sha256.Sum256([]byte(apiKey)))
					ctx = context.WithValue(ctx, apiKeyKey, true)
					ctx = context.WithValue(ctx, userKey, "api:"+hash[:16])
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// Try JWT from Authorization header
			tokenStr := ""
			if auth := r.Header.Get("Authorization"); auth != "" {
				parts := strings.SplitN(auth, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
					tokenStr = parts[1]
				}
			}

			// Fallback: read a_session cookie if no Authorization header
			if tokenStr == "" {
				if cookie, err := r.Cookie("a_session"); err == nil && cookie.Value != "" {
					tokenStr = cookie.Value
				}
			}

			// Fallback: read project-specific a_session_{projectID} cookie
			if tokenStr == "" {
				projectID := r.Header.Get("X-Applad-Project")
				if projectID != "" {
					if cookie, err := r.Cookie("a_session_" + projectID); err == nil && cookie.Value != "" {
						tokenStr = cookie.Value
					}
				}
			}

			if tokenStr != "" {
				token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
					if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
					}
					return []byte(jwtSecret), nil
				})
				if err == nil && token.Valid {
					if claims, ok := token.Claims.(*Claims); ok {
						ctx = context.WithValue(ctx, userKey, claims.Subject)
						ctx = context.WithValue(ctx, sessionKey, claims.SessionID)
						if claims.ProjectID != "" {
							ctx = context.WithValue(ctx, projectIDKey, claims.ProjectID)
						}
					}
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth is middleware that returns 401 if no auth credentials are present.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := UserFromContext(ctx)
		if userID == "" {
			apperr.Unauthorized(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}
