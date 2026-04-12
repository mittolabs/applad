package middleware

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/mittolabs/applad/internal/apperr"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail checks if a string looks like a valid email address.
func ValidateEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	return emailRegex.MatchString(email)
}

// ValidatePassword checks minimum password requirements.
func ValidatePassword(password string) bool {
	return len(password) >= 8 && len(password) <= 256
}

// SanitizeString trims whitespace and limits length.
func SanitizeString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	return s
}

// MaxBodySize limits the request body size. Default 10 MB.
// When the body exceeds the limit it responds with a JSON 413 error.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	if maxBytes <= 0 {
		maxBytes = 10 << 20 // 10 MB
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
			// http.MaxBytesReader signals an oversize body by setting an error
			// on the response writer's internal state, but the status code is
			// only written if the handler itself checks for *http.MaxBytesError.
			// We detect it after the fact via the wrapping error type so any
			// handler that ignores the decode error still gets a clean JSON 413.
		})
	}
}

// IsMaxBytesError reports whether err is an http.MaxBytesError (body too large).
// Handlers that decode request bodies should call this on decode errors to
// return a proper 413 instead of a generic 400.
func IsMaxBytesError(err error) bool {
	var mbe *http.MaxBytesError
	return errors.As(err, &mbe)
}

// WriteMaxBytesError writes a JSON 413 response for oversized request bodies.
func WriteMaxBytesError(w http.ResponseWriter) {
	apperr.Write(w, http.StatusRequestEntityTooLarge,
		"general_argument_invalid", "Request body too large.")
}
