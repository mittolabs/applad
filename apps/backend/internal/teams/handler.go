package teams

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
	"github.com/mittolabs/applad/internal/model"
)

// Handler handles HTTP requests for teams.
type Handler struct {
	svc *Service
}

// NewHandler creates a new teams Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// authorizeTeam decides whether the caller may act on teamID, and writes a 403
// and returns false when they may not. A server API key (which legitimately
// provisions teams) and a console admin keep full access. An end-user session
// must be a *joined* member of the team; when requireOwner is set they must
// additionally hold the "owner" role. This is what stops any authenticated
// project user from renaming/deleting another user's team, reading its roster,
// or self-adding to a privileged team — team roles feed database RLS, so an
// unchecked mutation here escalates into RLS-protected data.
func (h *Handler) authorizeTeam(w http.ResponseWriter, r *http.Request, teamID string, requireOwner bool) bool {
	if middleware.IsAPIKey(r.Context()) || middleware.IsConsoleAdmin(r.Context()) {
		return true
	}
	userID := middleware.UserFromContext(r.Context())
	if userID == "" {
		apperr.Write(w, http.StatusForbidden, "general_forbidden", "not authorized for this team")
		return false
	}
	member, owner, err := h.svc.MembershipOf(r.Context(), teamID, userID)
	if err != nil {
		apperr.Internal(w, err)
		return false
	}
	if !member || (requireOwner && !owner) {
		apperr.Write(w, http.StatusForbidden, "general_forbidden", "not authorized for this team")
		return false
	}
	return true
}

// Routes returns the teams router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.createTeam)
	r.Get("/", h.listTeams)
	r.Get("/{teamId}", h.getTeam)
	r.Put("/{teamId}", h.updateTeam)
	r.Put("/{teamId}/prefs", h.updatePrefs)
	r.Delete("/{teamId}", h.deleteTeam)
	r.Post("/{teamId}/memberships", h.createMembership)
	r.Get("/{teamId}/memberships", h.listMemberships)
	r.Get("/{teamId}/memberships/{membershipId}", h.getMembership)
	r.Patch("/{teamId}/memberships/{membershipId}/status", h.acceptMembership)
	r.Patch("/{teamId}/memberships/{membershipId}", h.updateMembership)
	r.Delete("/{teamId}/memberships/{membershipId}", h.deleteMembership)
	return r
}

func (h *Handler) createTeam(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		TeamID string   `json:"teamId"`
		Name   string   `json:"name"`
		Roles  []string `json:"roles"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	userID := middleware.UserFromContext(r.Context())
	t, err := h.svc.Create(r.Context(), projectID, body.TeamID, body.Name, userID, body.Roles)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) acceptMembership(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	membershipID := chi.URLParam(r, "membershipId")
	userID := middleware.UserFromContext(r.Context())
	var body struct {
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Secret == "" {
		apperr.BadRequest(w, "secret is required")
		return
	}
	m, err := h.svc.AcceptMembership(r.Context(), teamID, membershipID, userID, body.Secret)
	if err != nil {
		apperr.Write(w, http.StatusUnauthorized, "membership_invalid_secret", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) listTeams(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	pg := middleware.ParsePagination(r)
	search := r.URL.Query().Get("search")
	// A plain user sees only the teams they belong to; an admin or API key sees
	// the whole project. Scoping by membership is what stops one user learning
	// the names of everyone else's teams.
	userID := middleware.UserFromContext(r.Context())
	scoped := userID != "" && !middleware.IsAPIKey(r.Context()) && !middleware.IsConsoleAdmin(r.Context())
	var (
		teams []*model.Team
		total int
		err   error
	)
	if scoped {
		teams, total, err = h.svc.ListForUser(r.Context(), projectID, userID, pg.Limit, pg.Offset, search)
	} else {
		teams, total, err = h.svc.List(r.Context(), projectID, pg.Limit, pg.Offset, search)
	}
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if teams == nil {
		teams = []*model.Team{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "teams": teams})
}

func (h *Handler) getTeam(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	teamID := chi.URLParam(r, "teamId")
	t, err := h.svc.Get(r.Context(), teamID, projectID)
	if err != nil {
		apperr.NotFound(w, "team")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) updateTeam(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	teamID := chi.URLParam(r, "teamId")
	if !h.authorizeTeam(w, r, teamID, true) {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	t, err := h.svc.Update(r.Context(), teamID, projectID, body.Name)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) updatePrefs(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	teamID := chi.URLParam(r, "teamId")
	if !h.authorizeTeam(w, r, teamID, true) {
		return
	}
	var body struct {
		Prefs map[string]interface{} `json:"prefs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	t, err := h.svc.UpdatePrefs(r.Context(), teamID, projectID, body.Prefs)
	if err != nil {
		if err.Error() == "team not found" {
			apperr.NotFound(w, "team")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) deleteTeam(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	teamID := chi.URLParam(r, "teamId")
	if !h.authorizeTeam(w, r, teamID, true) {
		return
	}
	if err := h.svc.Delete(r.Context(), teamID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createMembership(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	teamID := chi.URLParam(r, "teamId")
	var body struct {
		Email string   `json:"email"`
		Roles []string `json:"roles"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		apperr.BadRequest(w, "email is required")
		return
	}
	// Inviting a member (and choosing its roles) is an owner-only act. Without
	// this a non-member could invite themselves with any roles, then accept the
	// returned secret and land inside a privileged team.
	if !h.authorizeTeam(w, r, teamID, true) {
		return
	}
	if body.Roles == nil {
		body.Roles = []string{}
	}
	m, err := h.svc.CreateMembership(r.Context(), teamID, projectID, body.Email, body.Roles)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (h *Handler) listMemberships(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	teamID := chi.URLParam(r, "teamId")
	// Only a joined member may see who else is on the team; a non-member gets a
	// 403 rather than the full roster and everyone's email.
	if !h.authorizeTeam(w, r, teamID, false) {
		return
	}
	memberships, total, err := h.svc.ListMemberships(r.Context(), teamID, projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if memberships == nil {
		memberships = []*model.Membership{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "memberships": memberships})
}

func (h *Handler) getMembership(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	membershipID := chi.URLParam(r, "membershipId")
	// A single membership carries a member's email and roles, so reading it is
	// gated the same as the roster: joined members only, never a non-member.
	if !h.authorizeTeam(w, r, teamID, false) {
		return
	}
	m, err := h.svc.GetMembership(r.Context(), teamID, membershipID)
	if err != nil {
		apperr.NotFound(w, "membership")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) updateMembership(w http.ResponseWriter, r *http.Request) {
	teamID := chi.URLParam(r, "teamId")
	membershipID := chi.URLParam(r, "membershipId")
	// Changing a member's roles feeds database RLS, so it is owner-only, the same
	// gate as inviting a member. This is also what stops a non-owner granting
	// themselves "owner" through this route.
	if !h.authorizeTeam(w, r, teamID, true) {
		return
	}
	var body struct {
		Roles []string `json:"roles"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	m, err := h.svc.UpdateMembershipRoles(r.Context(), teamID, membershipID, body.Roles)
	if err != nil {
		if err.Error() == "membership not found" {
			apperr.NotFound(w, "membership")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) deleteMembership(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	teamID := chi.URLParam(r, "teamId")
	membershipID := chi.URLParam(r, "membershipId")
	if !h.authorizeTeam(w, r, teamID, true) {
		return
	}
	if err := h.svc.DeleteMembership(r.Context(), membershipID, teamID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
