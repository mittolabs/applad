package status

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler serves the public status snapshot.
type Handler struct {
	svc *Service
}

// NewHandler creates a status Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the public status router (no auth).
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.get)
	return r
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	snap, err := h.svc.Snapshot(r.Context())
	if err != nil {
		http.Error(w, `{"error":"status unavailable"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	// Short cache so the page stays fresh but a burst of viewers doesn't hammer
	// the DB; also lets a CDN serve the last-known status during a blip.
	w.Header().Set("Cache-Control", "public, max-age=15")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(snap)
}
