package testlab

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/deploy"
	"github.com/mittolabs/applad/internal/middleware"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

// AIExplainer summarises a capture. Kept as a narrow interface so testlab does
// not depend on the AI package: the app wires the real one, a self-hosted build
// with no AI configured wires nothing and the feature is simply absent.
type AIExplainer interface {
	IsConfigured() bool
	Explain(ctx context.Context, system, user string) (string, error)
}

// Handler serves the test lab API.
type Handler struct {
	svc    *Service
	studio *Studio
	ai     AIExplainer
}

func NewHandler(svc *Service, q *queue.Queue, rdb *redis.Client) *Handler {
	return &Handler{svc: svc, studio: NewStudio(svc, q, rdb)}
}

// SetAI wires the AI explainer, if one is configured.
func (h *Handler) SetAI(ai AIExplainer) { h.ai = ai }

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// Routes returns the test lab router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	r.Route("/runners", func(r chi.Router) {
		r.Post("/", h.createSuite)
		r.Get("/", h.listSuites)
		r.Get("/{runnerId}", h.getSuite)
		r.Put("/{runnerId}", h.updateSuite)
		r.Delete("/{runnerId}", h.deleteSuite)
		r.Post("/{runnerId}/source", h.uploadSource)
		r.Post("/{runnerId}/run", h.triggerRun)
	})

	// The catalogue: one entry per behaviour, however it got there.
	r.Route("/tests", func(r chi.Router) {
		r.Get("/", h.listTests)
		r.Put("/{testId}/tags", h.setTags)
		r.Put("/{testId}/quarantine", h.setQuarantine)
		r.Get("/{testId}/history", h.testHistory)
	})

	// Selections: which tests, run when.
	r.Route("/suites", func(r chi.Router) {
		r.Post("/", h.createSelection)
		r.Get("/", h.listSelections)
		r.Put("/{suiteId}", h.updateSelection)
		r.Delete("/{suiteId}", h.deleteSelection)
		r.Post("/{suiteId}/run", h.runSelection)
	})

	r.Route("/runs", func(r chi.Router) {
		r.Get("/", h.listRuns)
		r.Get("/{runId}", h.getRun)
		r.Get("/{runId}/cases", h.listCases)
		r.Get("/{runId}/artifacts", h.listArtifacts)
		// Live output while a run is in progress.
		r.Get("/{runId}/stream", h.streamRun)
	})

	// Serving evidence: a recording is opened by a media element, so it is a
	// plain file response rather than JSON.
	r.Get("/artifacts/{artifactId}", h.getArtifact)

	return r
}

// ── Suites ──

func (h *Handler) createSuite(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body Runner
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

	runner, err := h.svc.CreateRunner(r.Context(), projectID, body)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, runner)
}

func (h *Handler) listSuites(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	runners, total, err := h.svc.ListRunners(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if runners == nil {
		runners = []*Runner{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "runners": runners})
}

func (h *Handler) getSuite(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	runner, err := h.svc.GetRunner(r.Context(), chi.URLParam(r, "runnerId"), projectID)
	if err != nil {
		apperr.NotFound(w, "runner")
		return
	}
	writeJSON(w, http.StatusOK, runner)
}

func (h *Handler) updateSuite(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body Runner
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if strings.TrimSpace(body.Command) == "" {
		apperr.BadRequest(w, "command is required")
		return
	}
	runner, err := h.svc.UpdateRunner(r.Context(), chi.URLParam(r, "runnerId"), projectID, body)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, runner)
}

func (h *Handler) deleteSuite(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	runnerID := chi.URLParam(r, "runnerId")
	if err := h.svc.DeleteRunner(r.Context(), runnerID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	// The uploaded source is keyed by suite and nothing else refers to it.
	os.Remove(deploy.SourceArchivePath(runnerID)) //nolint:errcheck
	w.WriteHeader(http.StatusNoContent)
}

// uploadSource accepts a gzipped tar or zip of the project to test, for suites
// that are not pulled from git. It shares the deploy module's storage layout,
// since the build worker extracts both the same way.
func (h *Handler) uploadSource(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	runnerID := chi.URLParam(r, "runnerId")

	if _, err := h.svc.GetRunner(r.Context(), runnerID, projectID); err != nil {
		apperr.NotFound(w, "suite")
		return
	}
	if err := os.MkdirAll(deploy.SourceDir(), 0o755); err != nil {
		apperr.Internal(w, err)
		return
	}

	dest := deploy.SourceArchivePath(runnerID)
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
	writeJSON(w, http.StatusCreated, map[string]interface{}{"runnerId": runnerID, "bytes": written})
}

// ── Runs ──

func (h *Handler) triggerRun(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		TriggerType string `json:"triggerType"`
		Actor       string `json:"actor"`
		SuiteID     string `json:"suiteId"`
		Target      string `json:"target"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck

	run, err := h.svc.Trigger(r.Context(), chi.URLParam(r, "runnerId"), projectID,
		body.TriggerType, body.Actor, TriggerOptions{SuiteID: body.SuiteID, Target: body.Target})
	if err != nil {
		apperr.NotFound(w, "runner")
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}

func (h *Handler) listTests(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	tests, err := h.svc.ListTests(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if tests == nil {
		tests = []*Test{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(tests), "tests": tests})
}

func (h *Handler) setTags(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if err := h.svc.SetTags(r.Context(), chi.URLParam(r, "testId"), projectID, body.Tags); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) setQuarantine(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Quarantined bool `json:"quarantined"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	if err := h.svc.SetQuarantined(r.Context(), chi.URLParam(r, "testId"), projectID, body.Quarantined); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createSelection(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body Selection
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required")
		return
	}
	sel, err := h.svc.CreateSelection(r.Context(), projectID, body)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sel)
}

func (h *Handler) listSelections(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	sels, err := h.svc.ListSelections(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if sels == nil {
		sels = []*Selection{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(sels), "suites": sels})
}

func (h *Handler) updateSelection(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body Selection
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	sel, err := h.svc.UpdateSelection(r.Context(), chi.URLParam(r, "suiteId"), projectID, body)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sel)
}

func (h *Handler) deleteSelection(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	if err := h.svc.DeleteSelection(r.Context(), chi.URLParam(r, "suiteId"), projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// runSelection runs a suite, optionally against a target other than its
// default — which is how the same suite checks main and a branch.
func (h *Handler) runSelection(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	suiteID := chi.URLParam(r, "suiteId")

	var body struct {
		Target string `json:"target"`
		Actor  string `json:"actor"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck

	sel, err := h.svc.GetSelection(r.Context(), suiteID, projectID)
	if err != nil {
		apperr.NotFound(w, "suite")
		return
	}
	runnerID := sel.RunnerID
	if runnerID == "" {
		runner, err := h.svc.RecordedRunner(r.Context(), projectID)
		if err != nil {
			apperr.Internal(w, err)
			return
		}
		runnerID = runner.ID
	}

	run, err := h.svc.Trigger(r.Context(), runnerID, projectID, "manual", body.Actor,
		TriggerOptions{SuiteID: suiteID, Target: body.Target})
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}

// testHistory returns what one test has done across runs, newest first, with
// the evidence each run left. Clicking a bar in the catalogue lands here.
func (h *Handler) testHistory(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	items, err := h.svc.TestHistory(r.Context(), chi.URLParam(r, "testId"), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if items == nil {
		items = []*HistoryEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(items), "history": items})
}

// streamRun forwards a run's output while it is still running, then closes.
func (h *Handler) streamRun(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	runID := chi.URLParam(r, "runId")
	if _, err := h.svc.GetRun(r.Context(), runID, projectID); err != nil {
		apperr.NotFound(w, "run")
		return
	}
	h.studio.StreamRunLogs(w, r, runID)
}

func (h *Handler) listRuns(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	runs, total, err := h.svc.ListRuns(r.Context(), projectID, r.URL.Query().Get("runnerId"), limit)
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

func (h *Handler) listArtifacts(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	items, err := h.svc.ListArtifacts(r.Context(), chi.URLParam(r, "runId"), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if items == nil {
		items = []*Artifact{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(items), "artifacts": items})
}

func (h *Handler) getArtifact(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	path, contentType, name, err := h.svc.OpenArtifact(r.Context(), chi.URLParam(r, "artifactId"), projectID)
	if err != nil {
		apperr.NotFound(w, "artifact")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		apperr.NotFound(w, "artifact")
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "inline; filename=\""+filepath.Base(name)+"\"")
	// ServeContent gives range requests, which a video player needs to seek.
	stat, _ := f.Stat()
	http.ServeContent(w, r, name, stat.ModTime(), f)
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
