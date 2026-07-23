package entitlements

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/policy"
)

// Handler serves the entitlements document and the capability registry.
type Handler struct{}

// NewHandler creates an entitlements Handler.
func NewHandler() *Handler { return &Handler{} }

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Routes returns the entitlements router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.get)
	r.Get("/capabilities", h.listCapabilities)
	return r
}

// get returns the entitlements for the requested subject. On a default install
// this is unlimited with no notices.
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("org")
	projectID := r.URL.Query().Get("project")
	if projectID == "" {
		// The project header is how the rest of the API scopes a request.
		projectID = r.Header.Get("X-Applad-Project")
	}
	writeJSON(w, http.StatusOK, Get(r.Context(), orgID, projectID))
}

// listCapabilities exposes the registry so a console build and the contract
// tests can check they agree with the server about what is gateable.
func (h *Handler) listCapabilities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"capabilities": policy.Capabilities(),
		"total":        len(policy.Capabilities()),
	})
}
