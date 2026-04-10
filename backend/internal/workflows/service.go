// Package workflows implements Applad's native workflow engine:
// workflow definitions, execution orchestration, and history.
package workflows

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/uid"
)

// --- Models ---

// Node represents a single step in a workflow.
type Node struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Label    string                 `json:"label"`
	Config   map[string]interface{} `json:"config"`
	Position map[string]float64     `json:"position,omitempty"`
}

// Edge connects two nodes in a workflow graph.
type Edge struct {
	ID        string `json:"id"`
	Source    string `json:"source"`
	Target   string `json:"target"`
	Condition string `json:"condition,omitempty"`
}

// Workflow is a complete workflow definition.
type Workflow struct {
	ID              string                 `json:"$id"`
	ProjectID       string                 `json:"projectId"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Status          string                 `json:"status"`
	TriggerType     string                 `json:"triggerType"`
	TriggerConfig   map[string]interface{} `json:"triggerConfig"`
	WebhookSecret   string                 `json:"webhookSecret,omitempty"` // returned only on create
	Nodes           []Node                 `json:"nodes"`
	Edges           []Edge                 `json:"edges"`
	ErrorWorkflowID string                 `json:"errorWorkflowId"`
	RetryAttempts   int                    `json:"retryAttempts"`
	RetryDelayMs    int                    `json:"retryDelayMs"`
	CreatedAt       time.Time              `json:"$createdAt"`
	UpdatedAt       time.Time              `json:"$updatedAt"`
}

// StepLog records the result of executing a single node.
type StepLog struct {
	NodeID     string      `json:"nodeId"`
	NodeType   string      `json:"nodeType"`
	Label      string      `json:"label"`
	Status     string      `json:"status"`
	Input      interface{} `json:"input"`
	Output     interface{} `json:"output"`
	Error      string      `json:"error,omitempty"`
	DurationMs int64       `json:"durationMs"`
}

// Execution records a single run of a workflow.
type Execution struct {
	ID          string                 `json:"$id"`
	WorkflowID  string                 `json:"workflowId"`
	ProjectID   string                 `json:"projectId"`
	Status      string                 `json:"status"`
	TriggerData map[string]interface{} `json:"triggerData"`
	StartedAt   *time.Time             `json:"startedAt"`
	CompletedAt *time.Time             `json:"completedAt"`
	DurationMs  int64                  `json:"durationMs"`
	Error       string                 `json:"error,omitempty"`
	Logs        []StepLog              `json:"logs"`
}

// --- Service ---

// Service handles workflow business logic.
type Service struct {
	db    *db.DB
	queue *queue.Queue // nil when used from the worker (no re-enqueue)
}

// NewService creates a new workflows Service.
func NewService(database *db.DB, q *queue.Queue) *Service {
	s := &Service{db: database, queue: q}

	// Register sub-workflow runner
	SubWorkflowRunner = func(ctx context.Context, workflowID string, triggerData map[string]interface{}, depth int) ([]StepLog, interface{}, error) {
		wf, err := s.GetByID(ctx, workflowID)
		if err != nil {
			return nil, nil, fmt.Errorf("sub-workflow %s not found: %w", workflowID, err)
		}
		triggerData["__depth__"] = depth
		logs, runErr := RunWorkflow(ctx, wf, triggerData)
		var lastOutput interface{}
		for i := len(logs) - 1; i >= 0; i-- {
			if logs[i].Status == "completed" && logs[i].Output != nil {
				lastOutput = logs[i].Output
				break
			}
		}
		return logs, lastOutput, runErr
	}

	return s
}

// Create creates a new workflow.
func (s *Service) Create(ctx context.Context, projectID, name, description, triggerType string, triggerConfig map[string]interface{}, nodes []Node, edges []Edge) (*Workflow, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()

	if triggerConfig == nil {
		triggerConfig = map[string]interface{}{}
	}
	if nodes == nil {
		nodes = []Node{}
	}
	if edges == nil {
		edges = []Edge{}
	}

	tcJSON, _ := json.Marshal(triggerConfig)
	nodesJSON, _ := json.Marshal(nodes)
	edgesJSON, _ := json.Marshal(edges)

	// Generate a webhook secret for webhook-triggered workflows.
	// Returned once on create; callers use it to verify X-Applad-Signature headers.
	var webhookSecret string
	if triggerType == "webhook" {
		b := make([]byte, 32)
		rand.Read(b) //nolint:errcheck
		webhookSecret = hex.EncodeToString(b)
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO workflows (id, project_id, name, description, status, trigger_type, trigger_config, webhook_secret, nodes, edges, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'draft', ?, ?, ?, ?, ?, ?, ?)`,
		id, projectID, name, description, triggerType, tcJSON, nullableString(webhookSecret), nodesJSON, edgesJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("workflows: create: %w", err)
	}

	return &Workflow{
		ID: id, ProjectID: projectID, Name: name, Description: description,
		Status: "draft", TriggerType: triggerType, TriggerConfig: triggerConfig,
		WebhookSecret: webhookSecret, // only populated on create response
		Nodes: nodes, Edges: edges, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// Get returns a workflow by ID.
func (s *Service) Get(ctx context.Context, id, projectID string) (*Workflow, error) {
	var w Workflow
	var tcJSON, nodesJSON, edgesJSON []byte
	var desc, errorWfID sql.NullString
	var retryAttempts, retryDelayMs sql.NullInt64

	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, description, status, trigger_type, trigger_config, nodes, edges, created_at, updated_at,
		        COALESCE(error_workflow_id, ''), COALESCE(retry_attempts, 0), COALESCE(retry_delay_ms, 0)
		 FROM workflows WHERE id = ? AND project_id = ?`, id, projectID,
	).Scan(&w.ID, &w.ProjectID, &w.Name, &desc, &w.Status, &w.TriggerType, &tcJSON, &nodesJSON, &edgesJSON, &w.CreatedAt, &w.UpdatedAt,
		&errorWfID, &retryAttempts, &retryDelayMs)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found")
	}
	if err != nil {
		return nil, err
	}

	w.Description = desc.String
	w.ErrorWorkflowID = errorWfID.String
	w.RetryAttempts = int(retryAttempts.Int64)
	w.RetryDelayMs = int(retryDelayMs.Int64)
	json.Unmarshal(tcJSON, &w.TriggerConfig)
	json.Unmarshal(nodesJSON, &w.Nodes)
	json.Unmarshal(edgesJSON, &w.Edges)
	if w.TriggerConfig == nil {
		w.TriggerConfig = map[string]interface{}{}
	}
	if w.Nodes == nil {
		w.Nodes = []Node{}
	}
	if w.Edges == nil {
		w.Edges = []Edge{}
	}
	return &w, nil
}

// GetByID returns a workflow by ID without project scoping (used by webhook trigger + worker).
func (s *Service) GetByID(ctx context.Context, id string) (*Workflow, error) {
	var w Workflow
	var tcJSON, nodesJSON, edgesJSON []byte
	var desc, errorWfID, webhookSecret sql.NullString
	var retryAttempts, retryDelayMs sql.NullInt64

	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, description, status, trigger_type, trigger_config, webhook_secret, nodes, edges, created_at, updated_at,
		        COALESCE(error_workflow_id, ''), COALESCE(retry_attempts, 0), COALESCE(retry_delay_ms, 0)
		 FROM workflows WHERE id = ?`, id,
	).Scan(&w.ID, &w.ProjectID, &w.Name, &desc, &w.Status, &w.TriggerType, &tcJSON, &webhookSecret, &nodesJSON, &edgesJSON, &w.CreatedAt, &w.UpdatedAt,
		&errorWfID, &retryAttempts, &retryDelayMs)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found")
	}
	if err != nil {
		return nil, err
	}

	w.Description = desc.String
	w.ErrorWorkflowID = errorWfID.String
	w.RetryAttempts = int(retryAttempts.Int64)
	w.RetryDelayMs = int(retryDelayMs.Int64)
	w.WebhookSecret = webhookSecret.String
	json.Unmarshal(tcJSON, &w.TriggerConfig)
	json.Unmarshal(nodesJSON, &w.Nodes)
	json.Unmarshal(edgesJSON, &w.Edges)
	if w.TriggerConfig == nil {
		w.TriggerConfig = map[string]interface{}{}
	}
	if w.Nodes == nil {
		w.Nodes = []Node{}
	}
	if w.Edges == nil {
		w.Edges = []Edge{}
	}
	return &w, nil
}

// List returns all workflows for a project.
func (s *Service) List(ctx context.Context, projectID string) ([]*Workflow, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, description, status, trigger_type, trigger_config, nodes, edges, created_at, updated_at,
		        COALESCE(error_workflow_id, ''), COALESCE(retry_attempts, 0), COALESCE(retry_delay_ms, 0)
		 FROM workflows WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var workflows []*Workflow
	for rows.Next() {
		var w Workflow
		var tcJSON, nodesJSON, edgesJSON []byte
		var desc, errorWfID sql.NullString
		var retryAttempts, retryDelayMs sql.NullInt64
		if err := rows.Scan(&w.ID, &w.ProjectID, &w.Name, &desc, &w.Status, &w.TriggerType, &tcJSON, &nodesJSON, &edgesJSON, &w.CreatedAt, &w.UpdatedAt,
			&errorWfID, &retryAttempts, &retryDelayMs); err != nil {
			return nil, 0, err
		}
		w.Description = desc.String
		w.ErrorWorkflowID = errorWfID.String
		w.RetryAttempts = int(retryAttempts.Int64)
		w.RetryDelayMs = int(retryDelayMs.Int64)
		json.Unmarshal(tcJSON, &w.TriggerConfig)
		json.Unmarshal(nodesJSON, &w.Nodes)
		json.Unmarshal(edgesJSON, &w.Edges)
		if w.TriggerConfig == nil {
			w.TriggerConfig = map[string]interface{}{}
		}
		if w.Nodes == nil {
			w.Nodes = []Node{}
		}
		if w.Edges == nil {
			w.Edges = []Edge{}
		}
		workflows = append(workflows, &w)
	}
	return workflows, len(workflows), nil
}

// Update updates a workflow.
func (s *Service) Update(ctx context.Context, id, projectID string, name, description, status, triggerType string, triggerConfig map[string]interface{}, nodes []Node, edges []Edge) (*Workflow, error) {
	tcJSON, _ := json.Marshal(triggerConfig)
	nodesJSON, _ := json.Marshal(nodes)
	edgesJSON, _ := json.Marshal(edges)

	_, err := s.db.ExecContext(ctx,
		`UPDATE workflows SET name=?, description=?, status=?, trigger_type=?, trigger_config=?, nodes=?, edges=?, updated_at=?
		 WHERE id=? AND project_id=?`,
		name, description, status, triggerType, tcJSON, nodesJSON, edgesJSON, time.Now().UTC(), id, projectID)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id, projectID)
}

// Delete removes a workflow and its executions (cascaded by FK).
func (s *Service) Delete(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM workflows WHERE id = ? AND project_id = ?", id, projectID)
	return err
}

// Execute triggers a workflow execution by creating a pending record and pushing to the queue.
func (s *Service) Execute(ctx context.Context, workflowID, projectID string, triggerData map[string]interface{}) (*Execution, error) {
	if triggerData == nil {
		triggerData = map[string]interface{}{}
	}

	execID := uid.New("unique()")
	tdJSON, _ := json.Marshal(triggerData)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO workflow_executions (id, workflow_id, project_id, status, trigger_data)
		 VALUES (?, ?, ?, 'pending', ?)`,
		execID, workflowID, projectID, tdJSON)
	if err != nil {
		return nil, fmt.Errorf("workflows: execute: %w", err)
	}

	// Enqueue for the executions worker
	if s.queue != nil {
		s.queue.Push(ctx, "executions", queue.Job{
			ID:   execID,
			Type: "workflow_execution",
			Payload: map[string]interface{}{
				"executionId": execID,
				"workflowId":  workflowID,
				"projectId":   projectID,
				"triggerData":  triggerData,
			},
			CreatedAt: time.Now().UTC(),
		})
	}

	return &Execution{
		ID: execID, WorkflowID: workflowID, ProjectID: projectID,
		Status: "pending", TriggerData: triggerData,
	}, nil
}

// GetExecution returns an execution by ID.
func (s *Service) GetExecution(ctx context.Context, executionID, workflowID, projectID string) (*Execution, error) {
	var e Execution
	var tdJSON, logsJSON []byte
	var errStr sql.NullString
	var startedAt, completedAt sql.NullTime

	err := s.db.QueryRowContext(ctx,
		`SELECT id, workflow_id, project_id, status, trigger_data, started_at, completed_at, duration_ms, error, logs
		 FROM workflow_executions WHERE id = ? AND workflow_id = ? AND project_id = ?`,
		executionID, workflowID, projectID,
	).Scan(&e.ID, &e.WorkflowID, &e.ProjectID, &e.Status, &tdJSON, &startedAt, &completedAt, &e.DurationMs, &errStr, &logsJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found")
	}
	if err != nil {
		return nil, err
	}

	if startedAt.Valid {
		e.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		e.CompletedAt = &completedAt.Time
	}
	e.Error = errStr.String
	json.Unmarshal(tdJSON, &e.TriggerData)
	json.Unmarshal(logsJSON, &e.Logs)
	if e.TriggerData == nil {
		e.TriggerData = map[string]interface{}{}
	}
	if e.Logs == nil {
		e.Logs = []StepLog{}
	}
	return &e, nil
}

// ListExecutions returns all executions for a workflow.
func (s *Service) ListExecutions(ctx context.Context, workflowID, projectID string) ([]*Execution, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, workflow_id, project_id, status, trigger_data, started_at, completed_at, duration_ms, error, logs
		 FROM workflow_executions WHERE workflow_id = ? AND project_id = ? ORDER BY COALESCE(started_at, '9999-12-31') DESC`,
		workflowID, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var execs []*Execution
	for rows.Next() {
		var e Execution
		var tdJSON, logsJSON []byte
		var errStr sql.NullString
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(&e.ID, &e.WorkflowID, &e.ProjectID, &e.Status, &tdJSON, &startedAt, &completedAt, &e.DurationMs, &errStr, &logsJSON); err != nil {
			return nil, 0, err
		}
		if startedAt.Valid {
			e.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			e.CompletedAt = &completedAt.Time
		}
		e.Error = errStr.String
		json.Unmarshal(tdJSON, &e.TriggerData)
		json.Unmarshal(logsJSON, &e.Logs)
		if e.TriggerData == nil {
			e.TriggerData = map[string]interface{}{}
		}
		if e.Logs == nil {
			e.Logs = []StepLog{}
		}
		execs = append(execs, &e)
	}
	return execs, len(execs), nil
}

// UpdateExecution updates the status, logs, error, and timing of an execution.
func (s *Service) UpdateExecution(ctx context.Context, executionID string, status string, startedAt *time.Time, completedAt *time.Time, durationMs int64, execErr string, logs []StepLog) error {
	logsJSON, _ := json.Marshal(logs)
	_, err := s.db.ExecContext(ctx,
		`UPDATE workflow_executions SET status=?, started_at=?, completed_at=?, duration_ms=?, error=?, logs=?
		 WHERE id=?`,
		status, startedAt, completedAt, durationMs, execErr, logsJSON, executionID)
	return err
}

// ── Workflow Versioning ──

// SaveVersion creates a snapshot of the current workflow state.
func (s *Service) SaveVersion(ctx context.Context, wf *Workflow, createdBy string) error {
	// Count existing versions
	var count int
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM workflow_versions WHERE workflow_id=?`, wf.ID).Scan(&count)

	nodesJSON, _ := json.Marshal(wf.Nodes)
	edgesJSON, _ := json.Marshal(wf.Edges)
	triggerJSON, _ := json.Marshal(wf.TriggerConfig)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO workflow_versions (id, workflow_id, version, name, description, nodes, edges, trigger_type, trigger_config, created_at, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uid.New("unique()"), wf.ID, count+1, wf.Name, wf.Description,
		nodesJSON, edgesJSON, wf.TriggerType, triggerJSON, time.Now().UTC(), createdBy)
	return err
}

// ListVersions returns all versions of a workflow.
func (s *Service) ListVersions(ctx context.Context, workflowID string) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, version, name, created_at, created_by FROM workflow_versions WHERE workflow_id=? ORDER BY version DESC`, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []map[string]interface{}
	for rows.Next() {
		var id, name string
		var version int
		var createdAt time.Time
		var createdBy sql.NullString
		rows.Scan(&id, &version, &name, &createdAt, &createdBy)
		versions = append(versions, map[string]interface{}{
			"$id": id, "version": version, "name": name,
			"$createdAt": createdAt, "createdBy": createdBy.String,
		})
	}
	return versions, nil
}

// ── Workflow Sharing ──

// ShareWorkflow shares a workflow with a user.
func (s *Service) ShareWorkflow(ctx context.Context, workflowID, userID, role string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO workflow_shares (id, workflow_id, user_id, role) VALUES (?, ?, ?, ?)
		 ON CONFLICT (workflow_id, user_id) DO UPDATE SET role=EXCLUDED.role`,
		uid.New("unique()"), workflowID, userID, role)
	return err
}

// UnshareWorkflow removes a share.
func (s *Service) UnshareWorkflow(ctx context.Context, workflowID, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM workflow_shares WHERE workflow_id=? AND user_id=?`, workflowID, userID)
	return err
}

// ListShares returns shares for a workflow.
func (s *Service) ListShares(ctx context.Context, workflowID string) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, role, created_at FROM workflow_shares WHERE workflow_id=?`, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []map[string]interface{}
	for rows.Next() {
		var id, userID, role string
		var createdAt time.Time
		rows.Scan(&id, &userID, &role, &createdAt)
		shares = append(shares, map[string]interface{}{
			"$id": id, "userId": userID, "role": role, "$createdAt": createdAt,
		})
	}
	return shares, nil
}

// ── Workflow Templates ──

// ListTemplates returns all available workflow templates.
func (s *Service) ListTemplates(ctx context.Context) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, description, category, icon, trigger_type, popularity FROM workflow_templates ORDER BY popularity DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []map[string]interface{}
	for rows.Next() {
		var id, name, category, icon, triggerType string
		var description sql.NullString
		var popularity int
		rows.Scan(&id, &name, &description, &category, &icon, &triggerType, &popularity)
		templates = append(templates, map[string]interface{}{
			"$id": id, "name": name, "description": description.String,
			"category": category, "icon": icon, "triggerType": triggerType,
			"popularity": popularity,
		})
	}
	return templates, nil
}

// GetTemplate returns a single template with full node/edge data.
func (s *Service) GetTemplate(ctx context.Context, templateID string) (*Workflow, error) {
	var name, triggerType string
	var description sql.NullString
	var nodesJSON, edgesJSON, triggerJSON []byte

	err := s.db.QueryRowContext(ctx,
		`SELECT name, description, trigger_type, trigger_config, nodes, edges FROM workflow_templates WHERE id=?`,
		templateID).Scan(&name, &description, &triggerType, &triggerJSON, &nodesJSON, &edgesJSON)
	if err != nil {
		return nil, err
	}

	wf := &Workflow{
		ID: templateID, Name: name, Description: description.String, TriggerType: triggerType,
	}
	json.Unmarshal(nodesJSON, &wf.Nodes)
	json.Unmarshal(edgesJSON, &wf.Edges)
	json.Unmarshal(triggerJSON, &wf.TriggerConfig)
	return wf, nil
}

// ── Workflow Folders ──

// CreateFolder creates a workflow folder.
func (s *Service) CreateFolder(ctx context.Context, projectID, name, parentID string) (string, error) {
	id := uid.New("unique()")
	var parent interface{} = nil
	if parentID != "" {
		parent = parentID
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO workflow_folders (id, project_id, name, parent_id) VALUES (?, ?, ?, ?)`,
		id, projectID, name, parent)
	return id, err
}

// ListFolders returns all folders for a project.
func (s *Service) ListFolders(ctx context.Context, projectID string) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, parent_id, created_at FROM workflow_folders WHERE project_id=? ORDER BY name`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []map[string]interface{}
	for rows.Next() {
		var id, name string
		var parentID sql.NullString
		var createdAt time.Time
		rows.Scan(&id, &name, &parentID, &createdAt)
		folders = append(folders, map[string]interface{}{
			"$id": id, "name": name, "parentId": parentID.String,
			"$createdAt": createdAt,
		})
	}
	return folders, nil
}
