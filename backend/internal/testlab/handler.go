package testlab

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/deploy"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler serves the test lab API.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// Routes returns the test lab router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	r.Route("/suites", func(r chi.Router) {
		r.Post("/", h.createSuite)
		r.Get("/", h.listSuites)
		r.Get("/{suiteId}", h.getSuite)
		r.Put("/{suiteId}", h.updateSuite)
		r.Delete("/{suiteId}", h.deleteSuite)
		r.Post("/{suiteId}/source", h.uploadSource)
		r.Post("/{suiteId}/run", h.triggerRun)
	})

	r.Route("/runs", func(r chi.Router) {
		r.Get("/", h.listRuns)
		r.Get("/{runId}", h.getRun)
		r.Get("/{runId}/cases", h.listCases)
	})

	return r
}

// ── Suites ──

func (h *Handler) createSuite(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body Suite
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	if strings.TrimSpace(body.Command) == "" {
		apperr.BadRequest(w, "command is required — the shell command that runs the suite")
		return
	}

	suite, err := h.svc.CreateSuite(r.Context(), projectID, body)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, suite)
}

func (h *Handler) listSuites(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	suites, total, err := h.svc.ListSuites(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if suites == nil {
		suites = []*Suite{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "suites": suites})
}

func (h *Handler) getSuite(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	suite, err := h.svc.GetSuite(r.Context(), chi.URLParam(r, "suiteId"), projectID)
	if err != nil {
		apperr.NotFound(w, "suite")
		return
	}
	writeJSON(w, http.StatusOK, suite)
}

func (h *Handler) updateSuite(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body Suite
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if strings.TrimSpace(body.Command) == "" {
		apperr.BadRequest(w, "command is required")
		return
	}
	suite, err := h.svc.UpdateSuite(r.Context(), chi.URLParam(r, "suiteId"), projectID, body)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, suite)
}

func (h *Handler) deleteSuite(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	suiteID := chi.URLParam(r, "suiteId")
	if err := h.svc.DeleteSuite(r.Context(), suiteID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	// The uploaded source is keyed by suite and nothing else refers to it.
	os.Remove(deploy.SourceArchivePath(suiteID)) //nolint:errcheck
	w.WriteHeader(http.StatusNoContent)
}

// uploadSource accepts a gzipped tar or zip of the project to test, for suites
// that are not pulled from git. It shares the deploy module's storage layout,
// since the build worker extracts both the same way.
func (h *Handler) uploadSource(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	suiteID := chi.URLParam(r, "suiteId")

	if _, err := h.svc.GetSuite(r.Context(), suiteID, projectID); err != nil {
		apperr.NotFound(w, "suite")
		return
	}
	if err := os.MkdirAll(deploy.SourceDir(), 0o755); err != nil {
		apperr.Internal(w, err)
		return
	}

	dest := deploy.SourceArchivePath(suiteID)
	f, err := os.Create(dest)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	defer f.Close()

	written, err := io.Copy(f, io.LimitReader(r.Body, 512<<20))
	if err != nil {
		os.Remove(dest) //nolint:errcheck
		apperr.Internal(w, err)
		return
	}
	if written == 0 {
		os.Remove(dest) //nolint:errcheck
		apperr.BadRequest(w, "empty source archive")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"suiteId": suiteID, "bytes": written})
}

// ── Runs ──

func (h *Handler) triggerRun(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		TriggerType string `json:"triggerType"`
		Actor       string `json:"actor"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck

	run, err := h.svc.Trigger(r.Context(), chi.URLParam(r, "suiteId"), projectID, body.TriggerType, body.Actor)
	if err != nil {
		apperr.NotFound(w, "suite")
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}

func (h *Handler) listRuns(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	runs, total, err := h.svc.ListRuns(r.Context(), projectID, r.URL.Query().Get("suiteId"), limit)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if runs == nil {
		runs = []*Run{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "runs": runs})
}

func (h *Handler) getRun(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	run, err := h.svc.GetRun(r.Context(), chi.URLParam(r, "runId"), projectID)
	if err != nil {
		apperr.NotFound(w, "run")
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (h *Handler) listCases(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	cases, total, err := h.svc.ListCases(r.Context(), chi.URLParam(r, "runId"), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if cases == nil {
		cases = []*CaseResult{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "cases": cases})
}
