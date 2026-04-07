package jobs

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
)

// Handler exposes the jobs HTTP API.
type Handler struct {
	svc *Service
}

// NewHandler creates a new jobs Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the jobs router.
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()

	// Queues
	r.Post("/queues", h.createQueue)
	r.Get("/queues", h.listQueues)
	r.Get("/queues/{queueId}", h.getQueue)
	r.Patch("/queues/{queueId}", h.updateQueue)
	r.Delete("/queues/{queueId}", h.deleteQueue)

	// Jobs within a queue
	r.Post("/queues/{queueId}/jobs", h.enqueue)
	r.Get("/queues/{queueId}/jobs", h.listJobs)
	r.Get("/queues/{queueId}/jobs/{jobId}", h.getJob)
	r.Delete("/queues/{queueId}/jobs/{jobId}", h.cancelJob)

	// Worker endpoints
	r.Post("/queues/{queueId}/dequeue", h.dequeue)
	r.Post("/jobs/{jobId}/ack", h.ack)
	r.Post("/jobs/{jobId}/nack", h.nack)

	return r
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// ── Queues ────────────────────────────────────────────────────────────────────

func (h *Handler) createQueue(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body struct {
		Name               string `json:"name"`
		WorkerURL          string `json:"workerUrl"`
		Concurrency        int    `json:"concurrency"`
		RetryLimit         int    `json:"retryLimit"`
		RetryDelayS        int    `json:"retryDelaySeconds"`
		DeadLetterQueueID  string `json:"deadLetterQueueId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "name is required")
		return
	}
	q, err := h.svc.CreateQueue(r.Context(), projectID, body.Name, body.WorkerURL,
		body.Concurrency, body.RetryLimit, body.RetryDelayS, body.DeadLetterQueueID)
	if err != nil {
		apperr.Write(w, http.StatusConflict, "queue_exists", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, q)
}

func (h *Handler) listQueues(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	queues, err := h.svc.ListQueues(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if queues == nil {
		queues = []*Queue{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": len(queues), "queues": queues})
}

func (h *Handler) getQueue(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	queueID := chi.URLParam(r, "queueId")
	q, err := h.svc.GetQueue(r.Context(), queueID, projectID)
	if err != nil {
		apperr.NotFound(w, "queue")
		return
	}
	q.Stats, _ = h.svc.queueStats(r.Context(), queueID)
	writeJSON(w, http.StatusOK, q)
}

func (h *Handler) updateQueue(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	queueID := chi.URLParam(r, "queueId")
	var body struct {
		Paused      *bool   `json:"paused"`
		Concurrency *int    `json:"concurrency"`
		RetryLimit  *int    `json:"retryLimit"`
		WorkerURL   *string `json:"workerUrl"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	q, err := h.svc.UpdateQueue(r.Context(), queueID, projectID, body.Paused, body.Concurrency, body.RetryLimit, body.WorkerURL)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, q)
}

func (h *Handler) deleteQueue(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	queueID := chi.URLParam(r, "queueId")
	if err := h.svc.DeleteQueue(r.Context(), queueID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Jobs ──────────────────────────────────────────────────────────────────────

func (h *Handler) enqueue(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	queueID := chi.URLParam(r, "queueId")
	var body struct {
		Name        string                 `json:"name"`
		Payload     map[string]interface{} `json:"payload"`
		Priority    int                    `json:"priority"`
		RunAt       string                 `json:"runAt"`
		MaxAttempts int                    `json:"maxAttempts"`
		DependsOn   []string               `json:"dependsOn"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		apperr.Write(w, http.StatusBadRequest, "general_argument_invalid", "name is required")
		return
	}
	runAt := time.Now().UTC()
	if body.RunAt != "" {
		if t, err := time.Parse(time.RFC3339, body.RunAt); err == nil {
			runAt = t
		}
	}
	j, err := h.svc.Enqueue(r.Context(), projectID, queueID, body.Name, body.Payload, body.Priority, runAt, body.MaxAttempts, body.DependsOn)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, j)
}

func (h *Handler) listJobs(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	queueID := chi.URLParam(r, "queueId")
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit == 0 {
		limit = 50
	}
	jobs, total, err := h.svc.ListJobs(r.Context(), queueID, projectID, q.Get("status"), limit, offset)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if jobs == nil {
		jobs = []*Job{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total": total, "limit": limit, "offset": offset, "jobs": jobs,
	})
}

func (h *Handler) getJob(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	jobID := chi.URLParam(r, "jobId")
	j, err := h.svc.GetJob(r.Context(), jobID, projectID)
	if err != nil {
		apperr.NotFound(w, "job")
		return
	}
	writeJSON(w, http.StatusOK, j)
}

func (h *Handler) cancelJob(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	jobID := chi.URLParam(r, "jobId")
	if err := h.svc.CancelJob(r.Context(), jobID, projectID); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Worker endpoints ──────────────────────────────────────────────────────────

func (h *Handler) dequeue(w http.ResponseWriter, r *http.Request) {
	queueID := chi.URLParam(r, "queueId")
	j, err := h.svc.Dequeue(r.Context(), queueID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if j == nil {
		writeJSON(w, http.StatusNoContent, nil)
		return
	}
	writeJSON(w, http.StatusOK, j)
}

func (h *Handler) ack(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")
	if err := h.svc.Ack(r.Context(), jobID); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "completed"})
}

func (h *Handler) nack(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")
	var body struct {
		Error string `json:"error"`
	}
	json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
	if err := h.svc.Nack(r.Context(), jobID, body.Error); err != nil {
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"requeued": true})
}
