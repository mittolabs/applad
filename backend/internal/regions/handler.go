package regions

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler exposes the regions HTTP API.
type Handler struct {
	svc *Service
}

// NewHandler creates a new regions Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// PublicRoutes returns the unauthenticated region catalog (list + get).
func PublicRoutes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/active", h.getActiveRegion)
	r.Get("/", h.listRegions)
	r.Put("/active", h.setActiveRegion)
	r.Get("/{regionId}/health", h.getHealth)
	r.Get("/{regionId}", h.getRegion)
	return r
}

// ProjectRoutes returns the per-project region assignment routes (require auth).
func ProjectRoutes(h *Handler) http.Handler {
	r := chi.NewRouter()
	// Catalog (duplicated here so project-scoped callers can also list)
	r.Get("/active", h.getActiveRegion)
	r.Get("/", h.listRegions)
	r.Put("/active", h.setActiveRegion)
	r.Get("/{regionId}/health", h.getHealth)
	r.Get("/{regionId}", h.getRegion)
	// Project assignments
	r.Get("/project", h.listProjectRegions)
	r.Post("/project", h.assignRegion)
	r.Delete("/project/{regionId}", h.removeRegion)
	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func (h *Handler) listRegions(w http.ResponseWriter, r *http.Request) {
	regions, err := h.svc.ListRegions(r.Context())
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if regions == nil {
		regions = []*Region{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(regions), "regions": regions})
}

func (h *Handler) getRegion(w http.ResponseWriter, r *http.Request) {
	regionID := chi.URLParam(r, "regionId")
	region, err := h.svc.GetRegion(r.Context(), regionID)
	if err != nil {
		apperr.NotFound(w, "region")
		return
	}
	writeJSON(w, http.StatusOK, region)
}

func (h *Handler) listProjectRegions(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	prs, err := h.svc.ListProjectRegions(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if prs == nil {
		prs = []*ProjectRegion{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(prs), "regions": prs})
}

func (h *Handler) assignRegion(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		RegionID string `json:"regionId"`
		Primary  bool   `json:"primary"`
		GDPR     bool   `json:"gdpr"`
		HIPAA    bool   `json:"hipaa"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RegionID == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "regionId is required")
		return
	}
	pr, err := h.svc.AssignRegion(r.Context(), projectID, body.RegionID, body.Primary, body.GDPR, body.HIPAA)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pr)
}

func (h *Handler) removeRegion(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	regionID := chi.URLParam(r, "regionId")
	if err := h.svc.RemoveRegion(r.Context(), projectID, regionID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getActiveRegion(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	if projectID == "" {
		projectID = r.Header.Get("X-Applad-Project")
	}
	if projectID == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "Missing X-Applad-Project header")
		return
	}
	pr, err := h.svc.GetPrimaryRegion(r.Context(), projectID)
	if err != nil {
		apperr.NotFound(w, "region")
		return
	}
	writeJSON(w, http.StatusOK, pr)
}

func (h *Handler) setActiveRegion(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	if projectID == "" {
		projectID = r.Header.Get("X-Applad-Project")
	}
	if projectID == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "Missing X-Applad-Project header")
		return
	}
	var body struct {
		RegionID string `json:"regionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RegionID == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "regionId is required")
		return
	}
	pr, err := h.svc.AssignRegion(r.Context(), projectID, body.RegionID, true, false, false)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pr)
}

func (h *Handler) getHealth(w http.ResponseWriter, r *http.Request) {
	regionID := chi.URLParam(r, "regionId")
	health, err := h.svc.RegionHealth(r.Context(), regionID)
	if err != nil {
		apperr.NotFound(w, "region")
		return
	}
	writeJSON(w, http.StatusOK, health)
}
