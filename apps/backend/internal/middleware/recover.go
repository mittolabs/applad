package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/mittolabs/applad/internal/apperr"
)

// Recover is a panic-recovery middleware that always responds with JSON.
// It replaces chi/middleware.Recoverer, which writes a plain-text stack trace.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				// Log the stack trace to stderr for operator visibility, but
				// never expose it to the client.
				fmt.Printf("panic: %v\n%s\n", rec, debug.Stack())
				apperr.Write(w, http.StatusInternalServerError,
					"general_server_error", "An unexpected error occurred.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
