package middleware

import "net/http"

// SecurityHeaders adds standard security headers to responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "1; mode=block")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=()")
		// Cache-Control is set per-endpoint; set a safe default for API responses.
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
