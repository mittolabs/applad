// Package health implements health check endpoints for all Applad dependencies.
package health

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/cache"
	"github.com/mittolabs/applad/internal/db"
)

// Handler handles health check requests.
type Handler struct {
	db    *db.DB
	cache *cache.Cache
}

// NewHandler creates a new health Handler.
func NewHandler(database *db.DB, cacheClient *cache.Cache) *Handler {
	return &Handler{db: database, cache: cacheClient}
}

type healthResponse struct {
	Status string `json:"status"`
	Ping   int64  `json:"ping,omitempty"`
}

func writeHealth(w http.ResponseWriter, status int, s string, ping int64) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(healthResponse{Status: s, Ping: ping})
}

// Routes returns the health check router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.server)
	r.Get("/db", h.dbCheck)
	r.Get("/cache", h.cacheCheck)
	return r
}

func (h *Handler) server(w http.ResponseWriter, r *http.Request) {
	writeHealth(w, http.StatusOK, "pass", 0)
}

func (h *Handler) dbCheck(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if err := h.db.PingContext(r.Context()); err != nil {
		writeHealth(w, http.StatusInternalServerError, "fail", 0)
		return
	}
	writeHealth(w, http.StatusOK, "pass", time.Since(start).Milliseconds())
}

func (h *Handler) cacheCheck(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if err := h.cache.Ping(r.Context()); err != nil {
		writeHealth(w, http.StatusInternalServerError, "fail", 0)
		return
	}
	writeHealth(w, http.StatusOK, "pass", time.Since(start).Milliseconds())
}
