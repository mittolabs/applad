package testlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// StudioRoutes serves recording sessions and the flows they produce.
func StudioRoutes(h *Handler) http.Handler {
	r := chi.NewRouter()

	r.Post("/sessions", h.startSession)
	r.Get("/sessions/{sessionId}", h.getSession)
	r.Delete("/sessions/{sessionId}", h.stopSession)
	// The live connection: frames and steps out, clicks and keys in.
	r.Get("/sessions/{sessionId}/stream", h.streamSession)
	// Saving turns the recording into a flow and a suite that can run it.
	r.Post("/sessions/{sessionId}/save", h.saveSession)
	r.Post("/sessions/{sessionId}/preview", h.previewSession)

	r.Get("/flows", h.listFlows)
	r.Get("/flows/{flowId}", h.getFlow)
	r.Delete("/flows/{flowId}", h.deleteFlow)

	return r
}

func (h *Handler) startSession(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Target string `json:"target"`
		Image  string `json:"image"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Target) == "" {
		apperr.BadRequest(w, "target is required — the URL to record against")
		return
	}

	sess, err := h.studio.Start(r.Context(), projectID, strings.TrimSpace(body.Target), body.Image)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sess)
}

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	sess, ok := h.studio.Get(chi.URLParam(r, "sessionId"), projectID)
	if !ok {
		apperr.NotFound(w, "session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"session": sess, "steps": sess.Steps()})
}

func (h *Handler) stopSession(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	h.studio.Stop(chi.URLParam(r, "sessionId"), projectID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) streamSession(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	sess, ok := h.studio.Get(chi.URLParam(r, "sessionId"), projectID)
	if !ok {
		apperr.NotFound(w, "session")
		return
	}
	h.studio.Stream(w, r, sess)
}

// previewSession shows what the recording would compile to, so somebody can
// read the test before deciding to keep it.
func (h *Handler) previewSession(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	sess, ok := h.studio.Get(chi.URLParam(r, "sessionId"), projectID)
	if !ok {
		apperr.NotFound(w, "session")
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	if body.Name == "" {
		body.Name = "recorded flow"
	}

	flow := Flow{Name: body.Name, Platform: "web", Target: sess.Target, Steps: sess.Steps()}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"playwright": CompilePlaywright(flow),
		"maestro":    CompileMaestro(flow, "com.example.app"),
	})
}

// saveSession keeps a recording: the flow as data, and a suite that runs it.
func (h *Handler) saveSession(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	sessionID := chi.URLParam(r, "sessionId")
	sess, ok := h.studio.Get(sessionID, projectID)
	if !ok {
		apperr.NotFound(w, "session")
		return
	}

	var body struct {
		Name string `json:"name"`
		// KeepOpen leaves the browser running, for recording several flows
		// from one session.
		KeepOpen bool `json:"keepOpen"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	if strings.TrimSpace(body.Name) == "" {
		apperr.BadRequest(w, "name is required — what this flow checks")
		return
	}

	steps := sess.Steps()
	if len(steps) <= 1 {
		apperr.BadRequest(w, "nothing recorded yet — interact with the page first")
		return
	}

	flow, err := h.svc.SaveFlow(r.Context(), projectID, Flow{
		Name: strings.TrimSpace(body.Name), Platform: "web", Target: sess.Target, Steps: steps,
	})
	if err != nil {
		apperr.Internal(w, err)
		return
	}

	if !body.KeepOpen {
		h.studio.Stop(sessionID, projectID)
	}
	writeJSON(w, http.StatusCreated, flow)
}

func (h *Handler) listFlows(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	flows, err := h.svc.ListFlows(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if flows == nil {
		flows = []*Flow{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(flows), "flows": flows})
}

func (h *Handler) getFlow(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	flow, err := h.svc.GetFlow(r.Context(), chi.URLParam(r, "flowId"), projectID)
	if err != nil {
		apperr.NotFound(w, "flow")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"flow":       flow,
		"playwright": CompilePlaywright(*flow),
	})
}

func (h *Handler) deleteFlow(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	if err := h.svc.DeleteFlow(r.Context(), chi.URLParam(r, "flowId"), projectID); err != nil {
		apperr.Internal(w, fmt.Errorf("testlab: delete flow: %w", err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
