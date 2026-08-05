package endpoints

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mittolabs/applad/internal/apperr"
	"github.com/mittolabs/applad/internal/middleware"
	"github.com/mittolabs/applad/internal/workflows"
)

// executionTimeout bounds a single synchronous endpoint run so a slow endpoint
// cannot pin a request goroutine indefinitely. Overridable via env.
var executionTimeout = func() time.Duration {
	if v := os.Getenv("ENDPOINT_EXECUTION_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 15 * time.Second
}()

// maxRequestBody caps the request body an endpoint will parse.
const maxRequestBody = 1 << 20 // 1MB

// execSlots bounds how many endpoint executions run at once. Endpoints run
// synchronously in the API process and hold a database connection while a data
// node runs, so an unbounded flood of slow endpoints could exhaust API
// goroutines or the pgbouncer pool and starve the rest of the API. This caps
// that blast radius; a request that cannot get a slot within a short wait is
// told to retry rather than piling on.
var execSlots = make(chan struct{}, endpointConcurrency())

func endpointConcurrency() int {
	if v := os.Getenv("ENDPOINT_MAX_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 64
}

// Handler serves both the authed management API and the public execution router.
type Handler struct {
	svc *Service
}

// NewHandler creates an endpoints Handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// Routes returns the authenticated management router (mounted at /endpoints).
func Routes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Get("/{endpointId}", h.get)
	r.Put("/{endpointId}", h.update)
	r.Delete("/{endpointId}", h.delete)
	r.Post("/{endpointId}/test", h.test)
	r.Get("/{endpointId}/executions", h.listExecutions)
	return r
}

// ExecuteRoutes returns the public execution router (mounted at /e). It matches
// any method; the endpoint's own auth field decides who may call it.
func ExecuteRoutes(h *Handler) http.Handler {
	r := chi.NewRouter()
	r.HandleFunc("/*", h.serve)
	return r
}

type endpointBody struct {
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Auth        string                 `json:"auth"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Nodes       []workflows.Node       `json:"nodes"`
	Edges       []workflows.Edge       `json:"edges"`
	Status      string                 `json:"status"`
}

func (b endpointBody) toEndpoint(projectID string) *Endpoint {
	return &Endpoint{
		ProjectID: projectID, Method: b.Method, Path: b.Path, Name: b.Name,
		Description: b.Description, Auth: b.Auth, InputSchema: b.InputSchema,
		Nodes: b.Nodes, Edges: b.Edges, Status: b.Status,
	}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	var body endpointBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Path) == "" {
		apperr.BadRequest(w, "path is required")
		return
	}
	if workflows.HasCycle(body.Nodes, body.Edges) {
		apperr.BadRequest(w, "endpoint graph must be acyclic")
		return
	}
	ep, err := h.svc.Create(r.Context(), body.toEndpoint(projectID))
	if err != nil {
		if err == ErrPathTaken {
			apperr.Write(w, http.StatusConflict, "endpoint_path_taken", err.Error())
			return
		}
		apperr.Internal(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ep)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	eps, total, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if eps == nil {
		eps = []*Endpoint{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"total": total, "endpoints": eps})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	ep, err := h.svc.Get(r.Context(), chi.URLParam(r, "endpointId"), middleware.ProjectFromContext(r.Context()))
	if err != nil {
		apperr.NotFound(w, "endpoint")
		return
	}
	writeJSON(w, http.StatusOK, ep)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "endpointId")
	var body endpointBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apperr.BadRequest(w, "invalid request body")
		return
	}
	if workflows.HasCycle(body.Nodes, body.Edges) {
		apperr.BadRequest(w, "endpoint graph must be acyclic")
		return
	}
	ep, err := h.svc.Update(r.Context(), id, projectID, body.toEndpoint(projectID))
	if err != nil {
		if err == ErrPathTaken {
			apperr.Write(w, http.StatusConflict, "endpoint_path_taken", err.Error())
			return
		}
		apperr.NotFound(w, "endpoint")
		return
	}
	writeJSON(w, http.StatusOK, ep)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "endpointId"), middleware.ProjectFromContext(r.Context())); err != nil {
		apperr.Internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listExecutions(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectFromContext(r.Context())
	id := chi.URLParam(r, "endpointId")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	execs, err := h.svc.ListExecutions(r.Context(), id, projectID, limit)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if execs == nil {
		execs = []*Execution{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"executions": execs})
}

// test runs an endpoint against a caller-supplied sample request, from the
// console. It runs regardless of draft/published status, under the calling
// user's identity, so the author sees exactly what a real call would do.
func (h *Handler) test(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	ep, err := h.svc.Get(ctx, chi.URLParam(r, "endpointId"), projectID)
	if err != nil {
		apperr.NotFound(w, "endpoint")
		return
	}
	var sample struct {
		Method  string                 `json:"method"`
		Path    string                 `json:"path"`
		Params  map[string]string      `json:"params"`
		Query   map[string]interface{} `json:"query"`
		Headers map[string]interface{} `json:"headers"`
		Body    interface{}            `json:"body"`
	}
	json.NewDecoder(r.Body).Decode(&sample) //nolint:errcheck

	method := sample.Method
	if method == "" {
		method = ep.Method
	}
	request := map[string]interface{}{
		"method":  method,
		"path":    firstNonEmpty(sample.Path, ep.Path),
		"params":  stringMapToIface(sample.Params),
		"query":   orEmptyMap(sample.Query),
		"headers": orEmptyMap(sample.Headers),
		"body":    sample.Body,
		"user":    middleware.UserFromContext(ctx),
		"project": projectID,
	}

	runCtx, cancel := context.WithTimeout(ctx, executionTimeout)
	defer cancel()
	res := h.svc.exe.Run(runCtx, ep, request, callContext{projectID: projectID, userID: middleware.UserFromContext(ctx)})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"statusCode": res.StatusCode,
		"body":       res.Body,
		"text":       res.Text,
		"isText":     res.IsText,
		"error":      res.Err,
		"logs":       res.Logs,
	})
}

// serve is the public execution entry point for /e/*.
func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	projectID := middleware.ProjectFromContext(ctx)
	if projectID == "" {
		apperr.Write(w, http.StatusBadRequest, "project_required", "An X-Applad-Project header is required.")
		return
	}
	path := "/" + strings.TrimPrefix(chi.URLParam(r, "*"), "/")

	ep, params, err := h.svc.MatchPublished(ctx, projectID, r.Method, path)
	if err != nil {
		apperr.Internal(w, err)
		return
	}
	if ep == nil {
		apperr.Write(w, http.StatusNotFound, "endpoint_not_found", "No published endpoint matches this method and path.")
		return
	}

	userID := middleware.UserFromContext(ctx)
	isKey := middleware.IsAPIKey(ctx)
	if !authorized(ep.Auth, userID, isKey) {
		apperr.Write(w, http.StatusUnauthorized, "endpoint_auth_required", "This endpoint requires authentication.")
		return
	}

	// Acquire an execution slot so a burst of slow endpoints cannot starve the
	// API. Wait briefly for transient contention, then shed load with a 503.
	select {
	case execSlots <- struct{}{}:
		defer func() { <-execSlots }()
	case <-time.After(2 * time.Second):
		w.Header().Set("Retry-After", "1")
		apperr.Write(w, http.StatusServiceUnavailable, "endpoint_busy", "The service is busy. Please retry shortly.")
		return
	case <-ctx.Done():
		return
	}

	request := map[string]interface{}{
		"method":  r.Method,
		"path":    path,
		"params":  stringMapToIface(params),
		"query":   flattenQuery(r),
		"headers": flattenHeaders(r),
		"body":    parseBody(r),
		"user":    userID,
		"session": middleware.SessionFromContext(ctx),
		"project": projectID,
	}

	runCtx, cancel := context.WithTimeout(ctx, executionTimeout)
	defer cancel()
	res := h.svc.exe.Run(runCtx, ep, request, callContext{projectID: projectID, userID: userID})

	// Record the run off the response's critical path, on a context that
	// outlives the (possibly cancelled) request. Best-effort.
	go h.recordExecution(ep, request, res)

	h.writeResult(w, res)
}

func (h *Handler) recordExecution(ep *Endpoint, request map[string]interface{}, res *Result) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	status := "ok"
	if res.Err != "" {
		status = "error"
	}
	var duration int64
	for _, l := range res.Logs {
		duration += l.DurationMs
	}
	var body interface{} = res.Body
	if res.IsText {
		body = res.Text
	}
	extra := handlerRedactSet(ep)
	h.svc.RecordExecution(ctx, &Execution{
		EndpointID: ep.ID, ProjectID: ep.ProjectID, Status: status,
		Method: firstNonEmpty(strFromMap(request, "method"), ep.Method),
		Path:   strFromMap(request, "path"), StatusCode: res.StatusCode,
		Request: redactedRequest(request, extra), Response: body,
		Logs: redactedLogs(res.Logs, extra), Error: res.Err, DurationMs: duration,
	})
}

// sensitiveKeyPattern matches body/field names whose values are secrets or
// regulated data. Their values are replaced with a marker in a stored record so
// a signup or payment endpoint does not build a durable plaintext store of
// passwords or card data, while the rest of the body stays visible for
// debugging.
var sensitiveKeyPattern = regexp.MustCompile(`(?i)pass|secret|token|api[-_]?key|authorization|cookie|cvv|card|ssn|credential`)

// redactValue deep-copies a JSON-ish value, replacing the values of keys that
// look sensitive (sensitiveKeyPattern) or that the author explicitly named in
// `extra` (lowercased). Non-object values pass through.
func redactValue(v interface{}, extra map[string]bool) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			if sensitiveKeyPattern.MatchString(k) || extra[strings.ToLower(k)] {
				out[k] = "[redacted]"
			} else {
				out[k] = redactValue(val, extra)
			}
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, val := range t {
			out[i] = redactValue(val, extra)
		}
		return out
	default:
		return v
	}
}

// redactedRequest returns a copy of the request safe to persist: the body is
// kept for debugging but its secret-named and author-named fields are redacted.
// Headers were already redacted at capture.
func redactedRequest(request map[string]interface{}, extra map[string]bool) map[string]interface{} {
	out := make(map[string]interface{}, len(request))
	for k, v := range request {
		if k == "body" {
			out[k] = redactValue(v, extra)
			continue
		}
		out[k] = v
	}
	return out
}

// redactedLogs copies the trace for storage, replacing the handler node's
// output (which is the whole request) with a redacted copy. The original logs
// returned to the author's test runner are left intact.
func redactedLogs(logs []workflows.StepLog, extra map[string]bool) []workflows.StepLog {
	out := make([]workflows.StepLog, len(logs))
	copy(out, logs)
	for i := range out {
		if out[i].NodeType == "endpoint_handler" {
			if req, ok := out[i].Output.(map[string]interface{}); ok {
				out[i].Output = redactedRequest(req, extra)
			}
		}
	}
	return out
}

// handlerRedactSet reads the author's explicit redaction list from the Request
// (handler) node's config, as a lowercased set of field names.
func handlerRedactSet(ep *Endpoint) map[string]bool {
	for i := range ep.Nodes {
		if ep.Nodes[i].Type != "endpoint_handler" {
			continue
		}
		list, ok := ep.Nodes[i].Config["redactFields"].([]interface{})
		if !ok {
			return nil
		}
		set := make(map[string]bool, len(list))
		for _, f := range list {
			if s, ok := f.(string); ok && strings.TrimSpace(s) != "" {
				set[strings.ToLower(strings.TrimSpace(s))] = true
			}
		}
		return set
	}
	return nil
}

func (h *Handler) writeResult(w http.ResponseWriter, res *Result) {
	if res.Err != "" && !res.Responded {
		status := res.StatusCode
		if status < 400 {
			status = http.StatusInternalServerError
		}
		// The public caller gets a generic message: a node's raw error can carry
		// SQL, table names, SQLSTATE and row-security policy detail, which must
		// not leak to an anonymous caller. The full error is still in the
		// recorded execution and the authenticated test runner, where the author
		// can debug it.
		writeJSON(w, status, map[string]interface{}{"error": publicErrorMessage(status)})
		return
	}
	if res.IsText {
		ct := res.ContentType
		if ct == "" {
			ct = "text/plain; charset=utf-8"
		}
		w.Header().Set("Content-Type", ct)
		w.WriteHeader(res.StatusCode)
		w.Write([]byte(res.Text)) //nolint:errcheck
		return
	}
	writeJSON(w, res.StatusCode, res.Body)
}

// publicErrorMessage is the caller-facing text for a failed run, chosen by
// status so it never carries internal detail.
func publicErrorMessage(status int) string {
	if status == http.StatusForbidden {
		return "You do not have permission to perform this action."
	}
	return "The endpoint could not be completed."
}

// authorized enforces the endpoint's call requirement.
func authorized(auth, userID string, isKey bool) bool {
	switch auth {
	case AuthSession:
		return userID != ""
	case AuthAPIKey:
		return isKey
	case AuthEither:
		return userID != "" || isKey
	default: // public
		return true
	}
}

// --- request helpers ---

func parseBody(r *http.Request) interface{} {
	if r.Body == nil {
		return nil
	}
	ct := r.Header.Get("Content-Type")
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody))
	if err != nil || len(raw) == 0 {
		return nil
	}
	if strings.Contains(ct, "application/json") {
		var parsed interface{}
		if json.Unmarshal(raw, &parsed) == nil {
			return parsed
		}
	}
	return string(raw)
}

func flattenQuery(r *http.Request) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range r.URL.Query() {
		if len(v) == 1 {
			out[k] = v[0]
		} else {
			out[k] = v
		}
	}
	return out
}

// sensitiveHeaders carry credentials and must never enter the endpoint graph or
// be persisted in an execution record. An endpoint reads the caller's identity
// through request.user / request.session, not the raw token, so redacting these
// costs nothing.
var sensitiveHeaders = map[string]bool{
	"authorization":       true,
	"cookie":              true,
	"set-cookie":          true,
	"x-applad-key":        true,
	"x-api-key":           true,
	"proxy-authorization": true,
}

func flattenHeaders(r *http.Request) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range r.Header {
		if sensitiveHeaders[strings.ToLower(k)] {
			out[k] = "[redacted]"
			continue
		}
		out[k] = strings.Join(v, ", ")
	}
	return out
}

func stringMapToIface(m map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func orEmptyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return map[string]interface{}{}
	}
	return m
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func strFromMap(m map[string]interface{}, key string) string {
	s, _ := m[key].(string)
	return s
}
