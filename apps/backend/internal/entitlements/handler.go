package entitlements

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
	"github.com/mittolabs/applad/internal/policy"
)

// OrgMemberChecker answers whether a console user belongs to an organization.
// Implemented by console.Service.
type OrgMemberChecker interface {
	IsOrgMember(ctx context.Context, userID, orgID string) (bool, error)
}

// Handler serves the entitlements document and the capability registry.
type Handler struct {
	dismissals *Dismissals
	members    OrgMemberChecker
}

// NewHandler creates an entitlements Handler.
func NewHandler(d *Dismissals, m OrgMemberChecker) *Handler {
	return &Handler{dismissals: d, members: m}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Routes returns the entitlements router. Mounted behind console auth: a notice
// can say something private ("your payment failed"), and limits describe an
// organization, so neither is public.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.get)
	r.Get("/capabilities", h.listCapabilities)
	r.Post("/notices/{id}/dismiss", h.dismiss)
	return r
}

// get returns the entitlements for the requested subject, minus anything this
// user has already dismissed.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.ConsoleUserFromContext(r.Context())
	orgID := r.URL.Query().Get("org")

	// Membership is checked rather than assumed: the org id is a caller-supplied
	// parameter, so without this anyone signed in could read any organization's
	// notices and limits by guessing one.
	if orgID != "" && h.members != nil {
		ok, err := h.members.IsOrgMember(r.Context(), userID, orgID)
		if err != nil || !ok {
			apperr.Write(w, http.StatusForbidden, "user_unauthorized",
				"You are not a member of this organization.")
			return
		}
	}

	projectID := r.URL.Query().Get("project")
	if projectID == "" {
		projectID = r.Header.Get("X-Applad-Project")
	}

	doc := Get(r.Context(), orgID, projectID)
	if h.dismissals != nil {
		doc = Filter(doc, h.dismissals.For(r.Context(), userID))
	}
	writeJSON(w, http.StatusOK, doc)
}

// dismiss clears one notice from this user's view, permanently. Per user, not
// per organization: everyone in the org was shown it, and each clears their own.
func (h *Handler) dismiss(w http.ResponseWriter, r *http.Request) {
	userID := middleware.ConsoleUserFromContext(r.Context())
	if userID == "" {
		apperr.Unauthorized(w)
		return
	}
	noticeID := chi.URLParam(r, "id")
	if noticeID == "" {
		apperr.BadRequest(w, "A notice id is required.")
		return
	}
	if h.dismissals != nil {
		if err := h.dismissals.Dismiss(r.Context(), userID, noticeID); err != nil {
			apperr.Internal(w, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// listCapabilities exposes the registry so a console build and the contract
// tests can check they agree with the server about what is gateable.
func (h *Handler) listCapabilities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"capabilities": policy.Capabilities(),
		"total":        len(policy.Capabilities()),
	})
}
