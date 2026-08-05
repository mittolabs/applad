// Package endpoints implements Applad's visual REST endpoint builder: a request
// enters at a handler node, flows through data/logic nodes, and terminates at a
// response node. Endpoints reuse the hardened workflow node executor and run
// synchronously in the API process (no container, no cold start). Own table,
// shared executor.
package endpoints

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
	"github.com/mittolabs/applad/internal/workflows"
)

// Auth requirement values for who may CALL an endpoint. Distinct from the
// per-node apply-rules toggle, which governs what the nodes may TOUCH.
const (
	AuthPublic  = "public"  // anyone
	AuthSession = "session" // a resolved user session
	AuthAPIKey  = "api_key" // a project API key
	AuthEither  = "either"  // a session OR an API key
)

// Endpoint is a method + path + graph, served synchronously.
type Endpoint struct {
	ID          string                 `json:"$id"`
	ProjectID   string                 `json:"projectId"`
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Auth        string                 `json:"auth"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Nodes       []workflows.Node       `json:"nodes"`
	Edges       []workflows.Edge       `json:"edges"`
	Status      string                 `json:"status"`
	Version     int                    `json:"version"`
	CreatedAt   time.Time              `json:"$createdAt"`
	UpdatedAt   time.Time              `json:"$updatedAt"`
}

// Execution records a single run of an endpoint.
type Execution struct {
	ID         string                 `json:"$id"`
	EndpointID string                 `json:"endpointId"`
	ProjectID  string                 `json:"projectId"`
	Status     string                 `json:"status"`
	Method     string                 `json:"method"`
	Path       string                 `json:"path"`
	StatusCode int                    `json:"statusCode"`
	Request    map[string]interface{} `json:"request"`
	Response   interface{}            `json:"response"`
	Logs       []workflows.StepLog    `json:"logs"`
	Error      string                 `json:"error,omitempty"`
	DurationMs int64                  `json:"durationMs"`
	CreatedAt  time.Time              `json:"$createdAt"`
}

// Service handles endpoint persistence and execution.
type Service struct {
	db  *db.DB
	exe *executor
}

// NewService creates an endpoints Service. dbSvc is the databases service the
// data nodes run their reads/writes against, so an endpoint's data operations
// go through the same RLS-safe path a client SDK does.
func NewService(database *db.DB, dbSvc DataService) *Service {
	return &Service{db: database, exe: newExecutor(dbSvc)}
}

func normalizeMethod(m string) string {
	m = strings.ToUpper(strings.TrimSpace(m))
	if m == "" {
		return "GET"
	}
	return m
}

// normalizePath forces a single leading slash and strips a trailing one (except
// the root), so "/users/{id}/" and "users/{id}" route identically.
func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if len(p) > 1 {
		p = strings.TrimRight(p, "/")
	}
	return p
}

func validAuth(a string) bool {
	switch a {
	case AuthPublic, AuthSession, AuthAPIKey, AuthEither:
		return true
	}
	return false
}

// Create inserts a new endpoint (as a draft).
func (s *Service) Create(ctx context.Context, e *Endpoint) (*Endpoint, error) {
	e.ID = uid.New("unique()")
	e.Method = normalizeMethod(e.Method)
	e.Path = normalizePath(e.Path)
	if !validAuth(e.Auth) {
		e.Auth = AuthPublic
	}
	if e.Status == "" {
		e.Status = "draft"
	}
	e.Version = 1
	now := time.Now().UTC()
	e.CreatedAt, e.UpdatedAt = now, now
	if e.Nodes == nil {
		e.Nodes = []workflows.Node{}
	}
	if e.Edges == nil {
		e.Edges = []workflows.Edge{}
	}

	schemaJSON, _ := json.Marshal(e.InputSchema)
	nodesJSON, _ := json.Marshal(e.Nodes)
	edgesJSON, _ := json.Marshal(e.Edges)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO endpoints (id, project_id, method, path, name, description, auth, input_schema, nodes, edges, status, version, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		e.ID, e.ProjectID, e.Method, e.Path, e.Name, e.Description, e.Auth, schemaJSON, nodesJSON, edgesJSON, e.Status, e.Version, now, now)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPathTaken
		}
		return nil, fmt.Errorf("endpoints: create: %w", err)
	}
	return e, nil
}

// ErrPathTaken is returned when a (project, method, path) already exists.
var ErrPathTaken = fmt.Errorf("an endpoint with this method and path already exists")

func isUniqueViolation(err error) bool {
	// pq/pgx surface 23505; match on the text to avoid a driver-specific import.
	return err != nil && strings.Contains(err.Error(), "23505")
}

const endpointColumns = `id, project_id, method, path, name, description, auth, input_schema, nodes, edges, status, version, created_at, updated_at`

func scanEndpoint(row interface{ Scan(...interface{}) error }) (*Endpoint, error) {
	var e Endpoint
	var desc sql.NullString
	var schemaJSON, nodesJSON, edgesJSON []byte
	if err := row.Scan(&e.ID, &e.ProjectID, &e.Method, &e.Path, &e.Name, &desc, &e.Auth,
		&schemaJSON, &nodesJSON, &edgesJSON, &e.Status, &e.Version, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return nil, err
	}
	e.Description = desc.String
	json.Unmarshal(schemaJSON, &e.InputSchema)
	json.Unmarshal(nodesJSON, &e.Nodes)
	json.Unmarshal(edgesJSON, &e.Edges)
	if e.Nodes == nil {
		e.Nodes = []workflows.Node{}
	}
	if e.Edges == nil {
		e.Edges = []workflows.Edge{}
	}
	return &e, nil
}

// Get returns an endpoint scoped to a project.
func (s *Service) Get(ctx context.Context, id, projectID string) (*Endpoint, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+endpointColumns+` FROM endpoints WHERE id=$1 AND project_id=$2`, id, projectID)
	e, err := scanEndpoint(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("endpoint not found")
	}
	return e, err
}

// List returns all endpoints for a project, newest first.
func (s *Service) List(ctx context.Context, projectID string) ([]*Endpoint, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+endpointColumns+` FROM endpoints WHERE project_id=$1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*Endpoint
	for rows.Next() {
		e, err := scanEndpoint(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, len(out), nil
}

// Update replaces an endpoint's definition.
func (s *Service) Update(ctx context.Context, id, projectID string, e *Endpoint) (*Endpoint, error) {
	e.Method = normalizeMethod(e.Method)
	e.Path = normalizePath(e.Path)
	if !validAuth(e.Auth) {
		e.Auth = AuthPublic
	}
	if e.Status == "" {
		e.Status = "draft"
	}
	if e.Nodes == nil {
		e.Nodes = []workflows.Node{}
	}
	if e.Edges == nil {
		e.Edges = []workflows.Edge{}
	}
	schemaJSON, _ := json.Marshal(e.InputSchema)
	nodesJSON, _ := json.Marshal(e.Nodes)
	edgesJSON, _ := json.Marshal(e.Edges)

	res, err := s.db.ExecContext(ctx,
		`UPDATE endpoints SET method=$1, path=$2, name=$3, description=$4, auth=$5, input_schema=$6, nodes=$7, edges=$8, status=$9,
		        version=version+1, updated_at=$10
		 WHERE id=$11 AND project_id=$12`,
		e.Method, e.Path, e.Name, e.Description, e.Auth, schemaJSON, nodesJSON, edgesJSON, e.Status, time.Now().UTC(), id, projectID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPathTaken
		}
		return nil, fmt.Errorf("endpoints: update: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, fmt.Errorf("endpoint not found")
	}
	return s.Get(ctx, id, projectID)
}

// Delete removes an endpoint (and its executions, via FK cascade).
func (s *Service) Delete(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM endpoints WHERE id=$1 AND project_id=$2`, id, projectID)
	return err
}

// MatchPublished finds the published endpoint for a project whose method + path
// template matches the incoming request, returning it and the extracted path
// params. A miss returns (nil, nil, nil).
func (s *Service) MatchPublished(ctx context.Context, projectID, method, path string) (*Endpoint, map[string]string, error) {
	method = normalizeMethod(method)
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+endpointColumns+` FROM endpoints WHERE project_id=$1 AND status='published' AND method=$2`, projectID, method)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	reqPath := normalizePath(path)
	// Exact-path templates should win over parameterised ones; collect and rank.
	var best *Endpoint
	var bestParams map[string]string
	bestScore := -1
	for rows.Next() {
		e, err := scanEndpoint(rows)
		if err != nil {
			return nil, nil, err
		}
		params, score, ok := matchPath(e.Path, reqPath)
		if ok && score > bestScore {
			best, bestParams, bestScore = e, params, score
		}
	}
	return best, bestParams, nil
}

// matchPath matches a stored path template ("/users/{id}") against a concrete
// request path ("/users/42"). It returns the captured params and a specificity
// score (higher = more literal segments) so an exact route beats a param route.
func matchPath(template, actual string) (map[string]string, int, bool) {
	tSeg := splitSegments(template)
	aSeg := splitSegments(actual)
	if len(tSeg) != len(aSeg) {
		return nil, 0, false
	}
	params := map[string]string{}
	score := 0
	for i, seg := range tSeg {
		if len(seg) >= 2 && seg[0] == '{' && seg[len(seg)-1] == '}' {
			params[seg[1:len(seg)-1]] = aSeg[i]
			continue
		}
		if seg != aSeg[i] {
			return nil, 0, false
		}
		score++
	}
	return params, score, true
}

func splitSegments(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return []string{}
	}
	return strings.Split(p, "/")
}

// RecordExecution persists a run record. Best-effort: a logging failure never
// affects the response the caller already received.
func (s *Service) RecordExecution(ctx context.Context, e *Execution) {
	e.ID = uid.New("unique()")
	reqJSON, _ := json.Marshal(e.Request)
	respJSON, _ := json.Marshal(e.Response)
	logsJSON, _ := json.Marshal(e.Logs)
	s.db.ExecContext(ctx, //nolint:errcheck
		`INSERT INTO endpoint_executions (id, endpoint_id, project_id, status, method, path, status_code, request, response, logs, error, duration_ms)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		e.ID, e.EndpointID, e.ProjectID, e.Status, e.Method, e.Path, e.StatusCode, reqJSON, respJSON, logsJSON, e.Error, e.DurationMs)
}

// ListExecutions returns recent runs of an endpoint (capped).
func (s *Service) ListExecutions(ctx context.Context, endpointID, projectID string, limit int) ([]*Execution, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, endpoint_id, project_id, status, method, path, status_code, request, response, logs, error, duration_ms, created_at
		 FROM endpoint_executions WHERE endpoint_id=$1 AND project_id=$2 ORDER BY created_at DESC LIMIT $3`,
		endpointID, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Execution
	for rows.Next() {
		var ex Execution
		var errStr sql.NullString
		var reqJSON, respJSON, logsJSON []byte
		if err := rows.Scan(&ex.ID, &ex.EndpointID, &ex.ProjectID, &ex.Status, &ex.Method, &ex.Path, &ex.StatusCode,
			&reqJSON, &respJSON, &logsJSON, &errStr, &ex.DurationMs, &ex.CreatedAt); err != nil {
			return nil, err
		}
		ex.Error = errStr.String
		json.Unmarshal(reqJSON, &ex.Request)
		json.Unmarshal(respJSON, &ex.Response)
		json.Unmarshal(logsJSON, &ex.Logs)
		out = append(out, &ex)
	}
	return out, nil
}
