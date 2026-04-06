// Package workflows implements Applad's native workflow engine:
// workflow definitions, execution orchestration, and history.
package workflows

import (
	"context"
	"database/sql"
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
	ID            string                 `json:"$id"`
	ProjectID     string                 `json:"projectId"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Status        string                 `json:"status"`
	TriggerType   string                 `json:"triggerType"`
	TriggerConfig map[string]interface{} `json:"triggerConfig"`
	Nodes         []Node                 `json:"nodes"`
	Edges         []Edge                 `json:"edges"`
	CreatedAt     time.Time              `json:"$createdAt"`
	UpdatedAt     time.Time              `json:"$updatedAt"`
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
	return &Service{db: database, queue: q}
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

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO workflows (id, project_id, name, description, status, trigger_type, trigger_config, nodes, edges, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'draft', ?, ?, ?, ?, ?, ?)`,
		id, projectID, name, description, triggerType, tcJSON, nodesJSON, edgesJSON, now, now)
	if err != nil {
		return nil, fmt.Errorf("workflows: create: %w", err)
	}

	return &Workflow{
		ID: id, ProjectID: projectID, Name: name, Description: description,
		Status: "draft", TriggerType: triggerType, TriggerConfig: triggerConfig,
		Nodes: nodes, Edges: edges, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// Get returns a workflow by ID.
func (s *Service) Get(ctx context.Context, id, projectID string) (*Workflow, error) {
	var w Workflow
	var tcJSON, nodesJSON, edgesJSON []byte
	var desc sql.NullString

	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, description, status, trigger_type, trigger_config, nodes, edges, created_at, updated_at
		 FROM workflows WHERE id = ? AND project_id = ?`, id, projectID,
	).Scan(&w.ID, &w.ProjectID, &w.Name, &desc, &w.Status, &w.TriggerType, &tcJSON, &nodesJSON, &edgesJSON, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found")
	}
	if err != nil {
		return nil, err
	}

	w.Description = desc.String
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
	var desc sql.NullString

	err := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, description, status, trigger_type, trigger_config, nodes, edges, created_at, updated_at
		 FROM workflows WHERE id = ?`, id,
	).Scan(&w.ID, &w.ProjectID, &w.Name, &desc, &w.Status, &w.TriggerType, &tcJSON, &nodesJSON, &edgesJSON, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workflow not found")
	}
	if err != nil {
		return nil, err
	}

	w.Description = desc.String
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
		`SELECT id, project_id, name, description, status, trigger_type, trigger_config, nodes, edges, created_at, updated_at
		 FROM workflows WHERE project_id = ? ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var workflows []*Workflow
	for rows.Next() {
		var w Workflow
		var tcJSON, nodesJSON, edgesJSON []byte
		var desc sql.NullString
		if err := rows.Scan(&w.ID, &w.ProjectID, &w.Name, &desc, &w.Status, &w.TriggerType, &tcJSON, &nodesJSON, &edgesJSON, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, 0, err
		}
		w.Description = desc.String
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
