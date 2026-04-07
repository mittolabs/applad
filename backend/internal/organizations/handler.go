package organizations

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
)

// Handler handles HTTP requests for organizations.
type Handler struct {
	svc *Service
}

// NewHandler creates a new organizations Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
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
	// Get user info from header (set by console auth middleware or passed from frontend)
	userID := r.Header.Get("X-Console-User-ID")
	email := r.Header.Get("X-Console-User-Email")
	name := r.Header.Get("X-Console-User-Name")

	org, err := h.svc.Create(r.Context(), body.Name, userID, email, name)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, org)
}

func (h *Handler) listByUser(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-Console-User-ID")
	if userID == "" {
		apperr.Unauthorized(w)
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
	id := chi.URLParam(r, "orgId")
	org, err := h.svc.Get(r.Context(), id)
	if err != nil {
		apperr.NotFound(w, "organization")
		return
	}
	writeJSON(w, http.StatusOK, org)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "orgId")
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
	id := chi.URLParam(r, "orgId")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgId")
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
	orgID := chi.URLParam(r, "orgId")
	var body struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		apperr.BadRequest(w, "email is required")
		return
	}
	member, token, err := h.svc.InviteMember(r.Context(), orgID, body.Email, body.Name)
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
	orgID := chi.URLParam(r, "orgId")
	memberID := chi.URLParam(r, "memberId")
	if err := h.svc.RemoveMember(r.Context(), orgID, memberID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) updateMemberRole(w http.ResponseWriter, r *http.Request) {
	orgID := chi.URLParam(r, "orgId")
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
	orgID := chi.URLParam(r, "orgId")
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
	orgID := chi.URLParam(r, "orgId")
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

func (h *Handler) acceptInvite(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	userID := r.Header.Get("X-Console-User-ID")
	if userID == "" {
		apperr.Unauthorized(w)
		return
	}
	if err := h.svc.AcceptInvite(r.Context(), token, userID); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}
