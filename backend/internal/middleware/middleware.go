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
	"github.com/mittolabs/applad/internal/model"
)

// contextKey is a typed key to avoid collisions.
type contextKey int

const (
	projectKey contextKey = iota
	userKey
	sessionKey
	apiKeyKey
	projectIDKey
	apiKeyScopesKey
	consoleAdminKey
)

// Claims is the JWT claims structure.
type Claims struct {
	jwt.RegisteredClaims
	SessionID string `json:"sid"`
	ProjectID string `json:"pid"`
	Console   bool   `json:"console"`
}

// ProjectProvider is implemented by the projects service.
type ProjectProvider interface {
	GetByKey(ctx context.Context, secret string) (interface{}, error)
	Get(ctx context.Context, id string) (interface{}, error)
}

// APIKeyProvider resolves raw API keys to stored key metadata.
type APIKeyProvider interface {
	GetKeyBySecret(ctx context.Context, secret string) (*model.APIKey, error)
}

// WriteJSON writes v as JSON to w.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
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

// APIKeyScopesFromContext returns scopes for the authenticated API key.
func APIKeyScopesFromContext(ctx context.Context) []string {
	v, _ := ctx.Value(apiKeyScopesKey).([]string)
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
func Authenticate(jwtSecret string, provider interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Try API key first
			if apiKey := r.Header.Get("X-Applad-Key"); apiKey != "" {
				// Validate format: applad_key_<hex>
				if strings.HasPrefix(apiKey, "applad_key_") {
					if keyProvider, ok := provider.(APIKeyProvider); ok {
						storedKey, err := keyProvider.GetKeyBySecret(ctx, apiKey)
						if err == nil {
							projectID := r.Header.Get("X-Applad-Project")
							if projectID != "" && storedKey.ProjectID != "" && storedKey.ProjectID != projectID {
								apperr.Write(w, http.StatusUnauthorized, "general_unauthorized_scope", "API key does not belong to the requested project.")
								return
							}
							if !apiKeyAllowed(storedKey.Scopes, r.Method, r.URL.Path) {
								apperr.Write(w, http.StatusForbidden, "permission_denied", "API key does not have the required scope for this resource.")
								return
							}
							ctx = context.WithValue(ctx, apiKeyScopesKey, storedKey.Scopes)
							ctx = context.WithValue(ctx, userKey, "api:"+storedKey.ID)
							ctx = context.WithValue(ctx, apiKeyKey, true)
							next.ServeHTTP(w, r.WithContext(ctx))
							return
						}
					}
					hash := fmt.Sprintf("%x", sha256.Sum256([]byte(apiKey)))
					ctx = context.WithValue(ctx, apiKeyKey, true)
					ctx = context.WithValue(ctx, apiKeyScopesKey, []string{})
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
						if claims.Console {
							// Console admin JWT — grant full access to all project resources.
							// userKey is intentionally left empty so service-layer RLS checks
							// are bypassed (same path as internal service calls).
							ctx = context.WithValue(ctx, consoleAdminKey, true)
						} else {
							// Validate that the JWT's project claim matches the request header
							// to prevent session fixation across projects.
							headerProjectID := r.Header.Get("X-Applad-Project")
							if claims.ProjectID != "" && headerProjectID != "" && claims.ProjectID != headerProjectID {
								next.ServeHTTP(w, r)
								return
							}
							ctx = context.WithValue(ctx, userKey, claims.Subject)
							ctx = context.WithValue(ctx, sessionKey, claims.SessionID)
							if claims.ProjectID != "" {
								ctx = context.WithValue(ctx, projectIDKey, claims.ProjectID)
							}
						}
					}
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func apiKeyAllowed(scopes []string, method, path string) bool {
	if len(scopes) == 0 {
		return false
	}
	required := apiKeyResourceScope(path)
	if required == "" {
		return true
	}
	isRead := method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
	for _, scope := range scopes {
		normalized := strings.TrimSpace(strings.ToLower(scope))
		switch normalized {
		case "*", "all":
			return true
		case "read":
			if isRead {
				return true
			}
		case required:
			return true
		case required + ".read", required + ":read":
			if isRead {
				return true
			}
		}
	}
	return false
}

func apiKeyResourceScope(path string) string {
	trimmed := strings.TrimPrefix(path, "/v1")
	switch {
	case strings.HasPrefix(trimmed, "/account"):
		return "auth"
	case strings.HasPrefix(trimmed, "/users"):
		return "users"
	case strings.HasPrefix(trimmed, "/teams"):
		return "teams"
	case strings.HasPrefix(trimmed, "/databases"):
		return "databases"
	case strings.HasPrefix(trimmed, "/storage"):
		return "storage"
	case strings.HasPrefix(trimmed, "/functions"):
		return "functions"
	case strings.HasPrefix(trimmed, "/messaging"):
		return "messaging"
	case strings.HasPrefix(trimmed, "/deploy"):
		return "deploy"
	case strings.HasPrefix(trimmed, "/workflows"):
		return "workflows"
	case strings.HasPrefix(trimmed, "/flags"):
		return "flags"
	case strings.HasPrefix(trimmed, "/analytics"):
		return "analytics"
	case strings.HasPrefix(trimmed, "/search"):
		return "search"
	case strings.HasPrefix(trimmed, "/vectors"):
		return "vectors"
	case strings.HasPrefix(trimmed, "/edge"):
		return "edge"
	case strings.HasPrefix(trimmed, "/billing"):
		return "billing"
	case strings.HasPrefix(trimmed, "/regions"):
		return "regions"
	default:
		return ""
	}
}

// RequireAuth is middleware that returns 401 if no auth credentials are present.
// Console admin JWTs (Console: true claim) are always permitted.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if IsConsoleAdmin(ctx) {
			next.ServeHTTP(w, r)
			return
		}
		userID := UserFromContext(ctx)
		if userID == "" {
			apperr.Unauthorized(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// IsConsoleAdmin returns true when the request was authenticated with a console admin JWT.
func IsConsoleAdmin(ctx context.Context) bool {
	v, _ := ctx.Value(consoleAdminKey).(bool)
	return v
}
