package teams

import (
	"encoding/json"
	"net/http"
	"strconv"
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

// Routes returns the teams router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.createTeam)
	r.Get("/", h.listTeams)
	r.Get("/{teamId}", h.getTeam)
	r.Put("/{teamId}", h.updateTeam)
	r.Delete("/{teamId}", h.deleteTeam)
	r.Post("/{teamId}/memberships", h.createMembership)
	r.Get("/{teamId}/memberships", h.listMemberships)
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
	t, err := h.svc.Create(r.Context(), projectID, body.TeamID, body.Name, body.Roles)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) listTeams(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	search := r.URL.Query().Get("search")
	teams, total, err := h.svc.List(r.Context(), projectID, limit, offset, search)
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

func (h *Handler) deleteTeam(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	teamID := chi.URLParam(r, "teamId")
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

func (h *Handler) deleteMembership(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	teamID := chi.URLParam(r, "teamId")
	membershipID := chi.URLParam(r, "membershipId")
	if err := h.svc.DeleteMembership(r.Context(), membershipID, teamID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
