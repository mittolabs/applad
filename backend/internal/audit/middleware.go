package audit

import (
	"net/http"
	"strings"

	"github.com/mittolabs/applad/internal/middleware"
)

// responseWriter captures the status code for audit logging.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Middleware returns HTTP middleware that records every authenticated API call.
func Middleware(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			projectID := middleware.ProjectFromContext(r.Context())
			if projectID == "" {
				return
			}

			// Derive resource type and action from path + method
			resourceType, resourceID, action := parseRoute(r.Method, r.URL.Path)

			svc.Record(r.Context(), Log{
				ProjectID:    projectID,
				UserID:       middleware.UserFromContext(r.Context()),
				Action:       action,
				ResourceType: resourceType,
				ResourceID:   resourceID,
				Method:       r.Method,
				Path:         r.URL.Path,
				StatusCode:   rw.statusCode,
				IPAddress:    realIP(r),
				UserAgent:    r.Header.Get("User-Agent"),
			})
		})
	}
}

// serviceNamespaces are top-level path prefixes that act as service containers
// rather than resource types with IDs following directly.
// e.g. /v1/storage/buckets/{id} — "storage" is the namespace, "buckets" is the resource.
var serviceNamespaces = map[string]bool{
	"storage": true,
}

// parseRoute extracts resource type, resource ID and a human-readable action
// from an HTTP method + path like "POST /v1/databases/abc/collections/xyz/documents".
func parseRoute(method, path string) (resourceType, resourceID, action string) {
	parts := strings.Split(strings.TrimPrefix(path, "/v1/"), "/")
	if len(parts) == 0 {
		return "unknown", "", method + " " + path
	}

	// If the first segment is a service namespace (e.g. "storage"), skip it so
	// the resource type starts at the next segment ("buckets", "files", etc.).
	start := 0
	if serviceNamespaces[parts[0]] {
		start = 1
		if start >= len(parts) {
			return parts[0], "", strings.ToLower(method) + "." + parts[0]
		}
	}

	resourceType = parts[start]

	// Walk path segments from start+1: odd offsets are IDs, even are sub-resources.
	// e.g. ["databases","dbId","collections","colId","documents","docId"]
	for i := start + 1; i < len(parts); i += 2 {
		if i+1 < len(parts) {
			resourceType = parts[i+1]
		} else {
			resourceID = parts[i]
		}
	}

	switch method {
	case http.MethodGet:
		if resourceID != "" {
			action = "read." + resourceType
		} else {
			action = "list." + resourceType
		}
	case http.MethodPost:
		action = "create." + resourceType
	case http.MethodPut, http.MethodPatch:
		action = "update." + resourceType
	case http.MethodDelete:
		action = "delete." + resourceType
	default:
		action = strings.ToLower(method) + "." + resourceType
	}
	return
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-Ip"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	return r.RemoteAddr
}
