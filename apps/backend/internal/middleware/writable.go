package middleware

import (
	"net/http"

	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/policy"
)

// EnforceWritable refuses mutating requests when the workspace is read-only.
//
// It is the single server-side site for the org.write capability, so a resolver
// can put a whole workspace read-only (a lapsed trial, say) without a check in
// every handler. Reads pass untouched — enforcement degrades access, it never
// traps data — and so does everything when no resolver is installed, which is
// the case for a self-hosted build. The project comes from ProjectContext, so
// this belongs after it in the chain.
func EnforceWritable(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			scope := policy.Project(ProjectFromContext(r.Context()))
			if err := policy.Require(r.Context(), "org.write", scope); err != nil {
				reason := "your workspace is read-only"
				if de, ok := err.(*policy.DeniedError); ok && de.Decision.Reason != "" {
					reason = de.Decision.Reason
				}
				apperr.Write(w, http.StatusPaymentRequired, "billing_read_only", reason)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
