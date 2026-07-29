package projects

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	mw "github.com/mittolabs/applad/internal/middleware"
	"github.com/mittolabs/applad/internal/model"
	oauthpkg "github.com/mittolabs/applad/internal/oauth"
)

// OAuthConfigStore is the per-project OAuth provider configuration store,
// satisfied by *oauth.ProjectOAuthService. Kept as an interface so the handler
// stays testable without a database.
type OAuthConfigStore interface {
	ListProviders(ctx context.Context, projectID string) ([]oauthpkg.ProjectOAuthConfig, error)
	SetProvider(ctx context.Context, projectID, provider, clientID, clientSecret string) error
	DisableProvider(ctx context.Context, projectID, provider string) error
	DeleteProvider(ctx context.Context, projectID, provider string) error
}

// AccessChecker decides what the signed-in console user may touch.
// Implemented by console.Service. The user id comes from the validated JWT in
// context, never from anything the client can set.
type AccessChecker interface {
	CanAccessProject(ctx context.Context, userID, projectID string) (bool, error)
	IsOrgMember(ctx context.Context, userID, orgID string) (bool, error)
	UserOrgIDs(ctx context.Context, userID string) ([]string, error)
	ProjectOrgs(ctx context.Context) (map[string]string, error)
	// DefaultOrg is the org a new project belongs to; "" when the user is in none.
	DefaultOrg(ctx context.Context, userID string) (string, error)
}

// Handler handles HTTP requests for project management.
type Handler struct {
	svc      *Service
	access   AccessChecker
	oauthCfg OAuthConfigStore
}

// NewHandler creates a new projects Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// SetAccess wires the ownership checker. Without it every handler denies —
// authorization fails closed rather than reverting to the open API this was.
func (h *Handler) SetAccess(a AccessChecker) {
	h.access = a
}

// SetOAuthConfig wires the per-project OAuth provider store. Without it the
// OAuth config routes return an error rather than silently succeeding.
func (h *Handler) SetOAuthConfig(s OAuthConfigStore) {
	h.oauthCfg = s
}

// callerID returns the console user id placed in context by RequireConsoleAuth,
// writing 401 when there is none.
func (h *Handler) callerID(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID := mw.ConsoleUserFromContext(r.Context())
	if userID == "" || h.access == nil {
		apperr.Unauthorized(w)
		return "", false
	}
	return userID, true
}

// requireProject rejects the request unless the caller may act on the project
// in the path. Unknown projects get the same 403, so probing ids reveals
// nothing; a DB error also denies.
func (h *Handler) requireProject(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, ok := h.callerID(w, r)
	if !ok {
		return "", false
	}
	projectID := chi.URLParam(r, "projectId")
	allowed, err := h.access.CanAccessProject(r.Context(), userID, projectID)
	if err != nil || !allowed {
		apperr.Write(w, http.StatusForbidden, "permission_denied",
			"You are not a member of the organization that owns this project.")
		return "", false
	}
	return projectID, true
}

// Routes returns the projects router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.createProject)
	r.Get("/", h.listProjects)
	r.Get("/{projectId}", h.getProject)
	r.Patch("/{projectId}", h.updateProject)
	r.Delete("/{projectId}", h.deleteProject)
	r.Post("/{projectId}/keys", h.createKey)
	r.Get("/{projectId}/keys", h.listKeys)
	r.Get("/{projectId}/keys/{keyId}", h.getKey)
	r.Patch("/{projectId}/keys/{keyId}", h.updateKey)
	r.Delete("/{projectId}/keys/{keyId}", h.deleteKey)
	r.Get("/{projectId}/usage", h.getUsage)
	r.Get("/{projectId}/search", h.search)
	r.Post("/{projectId}/platforms", h.createPlatform)
	r.Get("/{projectId}/platforms", h.listPlatforms)
	r.Get("/{projectId}/platforms/{platformId}", h.getPlatform)
	r.Patch("/{projectId}/platforms/{platformId}", h.updatePlatform)
	r.Delete("/{projectId}/platforms/{platformId}", h.deletePlatform)
	r.Patch("/{projectId}/auth", h.updateAuthConfig)
	r.Get("/{projectId}/auth/security", h.getAuthSecurity)
	r.Put("/{projectId}/auth/security", h.updateAuthSecurity)
	r.Get("/{projectId}/auth/oauth", h.listOAuthProviders)
	r.Put("/{projectId}/auth/oauth/{provider}", h.setOAuthProvider)
	r.Delete("/{projectId}/auth/oauth/{provider}", h.deleteOAuthProvider)
	r.Patch("/{projectId}/services", h.updateServicesConfig)
	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	userID, ok := h.callerID(w, r)
	if !ok {
		return
	}
	// A project must belong to an organization. Attach the caller's; refuse if
	// they are in none, rather than create a project owned by no one and
	// invisible to operators. Onboarding creates the org first, so this only
	// rejects a genuinely org-less caller.
	orgID, err := h.access.DefaultOrg(r.Context(), userID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if orgID == "" {
		apperr.Write(w, http.StatusConflict, "no_organization",
			"Create an organization before creating a project.")
		return
	}
	p, err := h.svc.Create(r.Context(), body.Name, body.Description, orgID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.callerID(w, r)
	if !ok {
		return
	}
	orgID := r.URL.Query().Get("orgId")

	var projects []*model.Project
	var err error
	if orgID != "" {
		// Asking for an org's projects requires being in that org.
		member, merr := h.access.IsOrgMember(r.Context(), userID, orgID)
		if merr != nil || !member {
			apperr.Write(w, http.StatusForbidden, "permission_denied", "You are not a member of this organization.")
			return
		}
		projects, err = h.svc.List(r.Context(), orgID)
	} else {
		// No org given: the caller sees their orgs' projects plus org-less
		// ones (pre-org/onboarding projects), never other tenants'.
		all, lerr := h.svc.List(r.Context())
		if lerr != nil {
			apperr.Internal(w, lerr)
			return
		}
		orgs, oerr := h.access.UserOrgIDs(r.Context(), userID)
		projOrgs, perr := h.access.ProjectOrgs(r.Context())
		if oerr != nil || perr != nil {
			apperr.Internal(w, fmt.Errorf("projects: resolve memberships"))
			return
		}
		mine := make(map[string]bool, len(orgs))
		for _, o := range orgs {
			mine[o] = true
		}
		projects = make([]*model.Project, 0, len(all))
		for _, p := range all {
			if org := projOrgs[p.ID]; org == "" || mine[org] {
				projects = append(projects, p)
			}
		}
	}
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":    len(projects),
		"projects": projects,
	})
}

func (h *Handler) getProject(w http.ResponseWriter, r *http.Request) {
	id, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	p, err := h.svc.Get(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "project")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) updateProject(w http.ResponseWriter, r *http.Request) {
	id, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	p, err := h.svc.Update(r.Context(), id, body.Name, body.Description)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "project")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) deleteProject(w http.ResponseWriter, r *http.Request) {
	id, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string   `json:"name"`
		Scopes    []string `json:"scopes"`
		ExpiresAt *string  `json:"expiresAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if body.Scopes == nil {
		body.Scopes = []string{}
	}
	var expiresAt *time.Time
	if body.ExpiresAt != nil && *body.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err != nil {
			apperr.BadRequest(w, "invalid expiresAt: use RFC3339 format")
			return
		}
		expiresAt = &t
	}
	// Ownership after validation: minting a key is full project access, so an
	// anonymous or foreign caller must never reach CreateKey.
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	key, _, err := h.svc.CreateKey(r.Context(), projectID, body.Name, body.Scopes, expiresAt)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, key)
}

func (h *Handler) listKeys(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	keys, err := h.svc.ListKeys(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total": len(keys),
		"keys":  keys,
	})
}

func (h *Handler) getKey(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	keyID := chi.URLParam(r, "keyId")
	key, err := h.svc.GetKey(r.Context(), projectID, keyID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "key")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, key)
}

func (h *Handler) updateKey(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	keyID := chi.URLParam(r, "keyId")
	var body struct {
		Name      *string  `json:"name"`
		Scopes    []string `json:"scopes"`
		ExpiresAt *string  `json:"expiresAt"` // RFC3339 or "" to clear
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	var expiresAt *time.Time
	if body.ExpiresAt != nil {
		if *body.ExpiresAt != "" {
			t, err := time.Parse(time.RFC3339, *body.ExpiresAt)
			if err != nil {
				apperr.BadRequest(w, "invalid expiresAt: use RFC3339 format")
				return
			}
			expiresAt = &t
		}
		// empty string = clear expiry (expiresAt stays nil)
	}
	key, err := h.svc.UpdateKey(r.Context(), projectID, keyID, body.Name, body.Scopes, body.ExpiresAt != nil, expiresAt)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			apperr.NotFound(w, "key")
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, key)
}

func (h *Handler) deleteKey(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	keyID := chi.URLParam(r, "keyId")
	if err := h.svc.DeleteKey(r.Context(), projectID, keyID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getUsage(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	usage, err := h.svc.GetUsage(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, usage)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"results": []*SearchResult{}})
		return
	}
	results, err := h.svc.Search(r.Context(), projectID, q, 20)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"results": results})
}

func (h *Handler) createPlatform(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	var body struct {
		Type     string `json:"type"`
		Name     string `json:"name"`
		Hostname string `json:"hostname"`
		StoreID  string `json:"storeId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.Type) == "" {
		apperr.BadRequest(w, "name and type are required")
		return
	}
	p, err := h.svc.CreatePlatform(r.Context(), projectID, body.Type, body.Name, body.Hostname, body.StoreID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) listPlatforms(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	platforms, err := h.svc.ListPlatforms(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":     len(platforms),
		"platforms": platforms,
	})
}

func (h *Handler) getPlatform(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	platformID := chi.URLParam(r, "platformId")
	p, err := h.svc.GetPlatform(r.Context(), projectID, platformID)
	if err != nil {
		apperr.NotFound(w, "platform")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) updatePlatform(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	platformID := chi.URLParam(r, "platformId")
	var body struct {
		Name           *string `json:"name"`
		Hostname       *string `json:"hostname"`
		DeployTargetID *string `json:"deployTargetId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	p, err := h.svc.UpdatePlatform(r.Context(), projectID, platformID, body.Name, body.Hostname, body.DeployTargetID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) deletePlatform(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	platformID := chi.URLParam(r, "platformId")
	if err := h.svc.DeletePlatform(r.Context(), projectID, platformID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) updateAuthConfig(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if err := h.svc.UpdateAuthConfig(r.Context(), projectID, body); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}

func (h *Handler) getAuthSecurity(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	sec, err := h.svc.GetAuthSecurity(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sec)
}

func (h *Handler) updateAuthSecurity(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	// Start from current settings so partial updates preserve other fields.
	sec, _ := h.svc.GetAuthSecurity(r.Context(), projectID)
	if err := json.NewDecoder(r.Body).Decode(&sec); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if err := h.svc.UpdateAuthSecurity(r.Context(), projectID, sec); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sec)
}

// listOAuthProviders returns the project's configured OAuth providers. The
// store's list query never selects the client secret, so a secret cannot leak
// here; only enabled + clientId are exposed.
func (h *Handler) listOAuthProviders(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	if h.oauthCfg == nil {
		apperr.Internal(w, fmt.Errorf("projects: oauth config store not configured"))
		return
	}
	cfgs, err := h.oauthCfg.ListProviders(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if cfgs == nil {
		cfgs = []oauthpkg.ProjectOAuthConfig{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total":     len(cfgs),
		"providers": cfgs,
	})
}

// setOAuthProvider creates or updates a single provider for the project. An
// empty clientSecret preserves the stored one (the GET never returns it), so a
// re-save that only changes the client id or toggles enabled does not wipe it.
func (h *Handler) setOAuthProvider(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	if h.oauthCfg == nil {
		apperr.Internal(w, fmt.Errorf("projects: oauth config store not configured"))
		return
	}
	provider := chi.URLParam(r, "provider")
	if _, known := oauthpkg.AllProviderDefinitions()[provider]; !known {
		apperr.BadRequest(w, "unsupported OAuth provider: "+provider)
		return
	}
	var body struct {
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
		Enabled      *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	enabled := body.Enabled == nil || *body.Enabled
	if enabled && strings.TrimSpace(body.ClientID) == "" {
		apperr.BadRequest(w, "clientId is required to enable a provider")
		return
	}
	// Upsert credentials, then flip enabled off when requested. SetProvider
	// always stores enabled=TRUE, so a disable is a second step.
	if err := h.oauthCfg.SetProvider(r.Context(), projectID, provider, strings.TrimSpace(body.ClientID), body.ClientSecret); err != nil {
		apperr.Internal(w, err)
		return
	}
	if !enabled {
		if err := h.oauthCfg.DisableProvider(r.Context(), projectID, provider); err != nil {
			apperr.Internal(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"provider": provider,
		"enabled":  enabled,
		"clientId": strings.TrimSpace(body.ClientID),
	})
}

// deleteOAuthProvider removes a provider configuration for the project.
func (h *Handler) deleteOAuthProvider(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	if h.oauthCfg == nil {
		apperr.Internal(w, fmt.Errorf("projects: oauth config store not configured"))
		return
	}
	provider := chi.URLParam(r, "provider")
	if err := h.oauthCfg.DeleteProvider(r.Context(), projectID, provider); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) updateServicesConfig(w http.ResponseWriter, r *http.Request) {
	projectID, ok := h.requireProject(w, r)
	if !ok {
		return
	}
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if err := h.svc.UpdateServicesConfig(r.Context(), projectID, body); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok"})
}
