package transfer

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler exposes the data-migration API under /v1/migrations.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Routes returns the migrations router. Mounted behind project + auth, so the
// caller is an authenticated actor in the target project.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Post("/validate", h.validate)
	// Cross-instance export: another Applad instance pulls this project as an
	// NDJSON stream (or ?report=1 for counts). Static route, so it takes
	// precedence over /{id}.
	r.Get("/export", h.export)
	r.Get("/{id}", h.get)
	r.Post("/{id}/cancel", h.cancel)
	return r
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"message": msg})
}

// projectID is resolved by ProjectContext middleware upstream.
func projectID(r *http.Request) string {
	return middleware.ProjectFromContext(r.Context())
}

type createReq struct {
	Source      string         `json:"source"`
	Groups      []Group        `json:"groups"`
	Options     map[string]any `json:"options"`
	Credentials map[string]any `json:"credentials"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	pid := projectID(r)
	if pid == "" {
		writeErr(w, http.StatusBadRequest, "missing project context")
		return
	}
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	m, err := h.svc.Create(r.Context(), pid, req.Source, req.Groups, req.Options, req.Credentials)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

// validate runs the pre-flight report: connect to the source and count what is
// available, without persisting anything.
func (h *Handler) validate(w http.ResponseWriter, r *http.Request) {
	if projectID(r) == "" {
		writeErr(w, http.StatusBadRequest, "missing project context")
		return
	}
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	counts, err := h.svc.Validate(r.Context(), req.Source, req.Groups, req.Credentials)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"source": req.Source, "counts": counts})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	pid := projectID(r)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	items, total, err := h.svc.List(r.Context(), pid, limit, offset)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if items == nil {
		items = []*Migration{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"total": total, "migrations": items})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	m, err := h.svc.Get(r.Context(), chi.URLParam(r, "id"), projectID(r))
	if err != nil {
		writeErr(w, http.StatusNotFound, "migration not found")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// export streams this project's resources as NDJSON to an authenticated caller
// (a destination Applad instance). With ?report=1 it returns per-group counts
// instead. Auth is the project API key enforced by the mounting middleware, so
// the caller must hold a key for this project — the same authorization the
// same-instance import path requires.
func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	pid := projectID(r)
	if pid == "" {
		writeErr(w, http.StatusBadRequest, "missing project context")
		return
	}
	// Export streams password hashes, so it is a server operation: require a
	// project API key, not an end-user session. This mirrors the same-instance
	// import, which also authorizes with an API key for the source project.
	if !middleware.IsAPIKey(r.Context()) {
		writeErr(w, http.StatusForbidden, "export requires a project API key")
		return
	}
	groups := parseGroups(r.URL.Query().Get("groups"))

	if r.URL.Query().Get("report") == "1" {
		counts, err := h.svc.ExportReport(r.Context(), pid, groups)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"counts": counts})
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	flusher, _ := w.(http.Flusher)

	emit := func(_ context.Context, res []Resource) error {
		for _, rr := range res {
			raw, err := json.Marshal(rr)
			if err != nil {
				return err
			}
			if err := enc.Encode(wireResource{Kind: rr.Kind(), Data: raw}); err != nil {
				return err
			}
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	}
	if err := h.svc.ExportStream(r.Context(), pid, groups, emit); err != nil {
		// The stream may be partly written; append a terminal error line so the
		// reader fails the migration rather than treating a truncated stream as
		// complete.
		msg, _ := json.Marshal(map[string]string{"message": err.Error()})
		_ = enc.Encode(wireResource{Kind: "error", Data: msg})
	}
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Cancel(r.Context(), chi.URLParam(r, "id"), projectID(r)); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}
