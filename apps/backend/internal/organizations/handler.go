package organizations

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	mw "github.com/mittolabs/applad/internal/middleware"
)

// AccessChecker answers who the signed-in console user is allowed to be here.
// Implemented by console.Service. The user id itself comes from the validated
// JWT in context — never from a client header, which anyone can set.
type AccessChecker interface {
	IsOrgMember(ctx context.Context, userID, orgID string) (bool, error)
	UserEmailName(ctx context.Context, userID string) (email, name string, err error)
}

// Handler handles HTTP requests for organizations.
type Handler struct {
	svc    *Service
	access AccessChecker
}

// NewHandler creates a new organizations Handler.
func NewHandler(svc *Service, access AccessChecker) *Handler {
	return &Handler{svc: svc, access: access}
}

// callerID returns the console user id placed in context by RequireConsoleAuth,
// writing 401 when there is none. Nil access also denies: fail closed.
func (h *Handler) callerID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := mw.ConsoleUserFromContext(r.Context())
	if userID == "" || h.access == nil {
		apperr.Unauthorized(w)
		return "", false
	}
	return userID, true
}

// requireMember rejects the request unless the caller is an active member of
// the org in the path. Unknown orgs get the same 403, so probing ids reveals
// nothing; a DB error also denies.
func (h *Handler) requireMember(w http.ResponseWriter, r *http.Request) (orgID, userID string, ok bool) {
	userID, ok = h.callerID(w, r)
	if !ok {
		return "", "", false
	}
	orgID = chi.URLParam(r, "orgId")
	member, err := h.access.IsOrgMember(r.Context(), userID, orgID)
	if err != nil || !member {
		apperr.Write(w, http.StatusForbidden, "permission_denied", "You are not a member of this organization.")
		return "", "", false
	}
	return orgID, userID, true
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Routes returns the organizations router (requires console JWT auth).
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.create)
	r.Get("/", h.listByUser)
	r.Get("/{orgId}", h.get)
	r.Patch("/{orgId}", h.update)
	r.Delete("/{orgId}", h.delete)
	r.Get("/{orgId}/members", h.listMembers)
	r.Post("/{orgId}/members", h.inviteMember)
	r.Delete("/{orgId}/members/{memberId}", h.removeMember)
	r.Patch("/{orgId}/members/{memberId}", h.updateMemberRole)
	r.Get("/{orgId}/projects", h.listProjects)
	r.Post("/{orgId}/projects", h.createProject)
	r.Get("/{orgId}/stats", h.getStats)
	r.Get("/{orgId}/activity", h.listActivity)
	r.Post("/invites/{token}/accept", h.acceptInvite)
	return r
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	// Identity comes from the validated JWT; email/name from the DB. The
	// headers this used to read were writable by any caller.
	userID, ok := h.callerID(w, r)
	if !ok {
		return
	}
	email, name, err := h.access.UserEmailName(r.Context(), userID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	org, err := h.svc.Create(r.Context(), body.Name, userID, email, name)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, org)
}

func (h *Handler) listByUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.callerID(w, r)
	if !ok {
		return
	}
	orgs, err := h.svc.ListByUser(r.Context(), userID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if orgs == nil {
		orgs = []*Organization{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(orgs), "organizations": orgs})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, _, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	org, err := h.svc.Get(r.Context(), id)
	if err != nil {
		apperr.NotFound(w, "organization")
		return
	}
	writeJSON(w, http.StatusOK, org)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, _, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	org, err := h.svc.Update(r.Context(), id, body.Name)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, org)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, _, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	orgID, _, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	members, err := h.svc.ListMembers(r.Context(), orgID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if members == nil {
		members = []*Member{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(members), "members": members})
}

func (h *Handler) inviteMember(w http.ResponseWriter, r *http.Request) {
	orgID, _, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	var body struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		apperr.BadRequest(w, "email is required")
		return
	}
	if body.Role == "" {
		body.Role = "member"
	}
	// Validate role
	switch body.Role {
	case "owner", "admin", "member":
		// valid
	default:
		apperr.BadRequest(w, "role must be owner, admin, or member")
		return
	}
	member, token, err := h.svc.InviteMember(r.Context(), orgID, body.Email, body.Name, body.Role)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"member":      member,
		"inviteToken": token,
	})
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	orgID, _, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	memberID := chi.URLParam(r, "memberId")
	if err := h.svc.RemoveMember(r.Context(), orgID, memberID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) updateMemberRole(w http.ResponseWriter, r *http.Request) {
	orgID, _, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	memberID := chi.URLParam(r, "memberId")
	var body struct {
		Role string `json:"role"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if err := h.svc.UpdateMemberRole(r.Context(), orgID, memberID, body.Role); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	orgID, _, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	projects, err := h.svc.ListProjects(r.Context(), orgID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if projects == nil {
		projects = []map[string]interface{}{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(projects), "projects": projects})
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	orgID, _, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "name is required")
		return
	}
	project, err := h.svc.CreateProject(r.Context(), orgID, body.Name, body.Description)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (h *Handler) getStats(w http.ResponseWriter, r *http.Request) {
	orgID, _, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	stats, err := h.svc.GetOrgStats(r.Context(), orgID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handler) listActivity(w http.ResponseWriter, r *http.Request) {
	orgID, _, ok := h.requireMember(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit == 0 {
		limit = 50
	}
	entries, total, err := h.svc.ListActivity(r.Context(), orgID, limit, offset)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":    total,
		"activity": entries,
	})
}

func (h *Handler) acceptInvite(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	// The invite is bound to the signed-in account, not to a caller-chosen id.
	userID, ok := h.callerID(w, r)
	if !ok {
		return
	}
	if err := h.svc.AcceptInvite(r.Context(), token, userID); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}
