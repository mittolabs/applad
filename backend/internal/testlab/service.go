package testlab

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

// Service manages test suites and their runs.
type Service struct {
	db    *db.DB
	queue *queue.Queue
}

func NewService(database *db.DB, q *queue.Queue) *Service {
	return &Service{db: database, queue: q}
}

// Suite is a project's configuration for running its tests.
type Suite struct {
	ID         string `json:"$id"`
	ProjectID  string `json:"projectId"`
	Name       string `json:"name"`
	SourceType string `json:"sourceType"`
	SourceURL  string `json:"sourceUrl,omitempty"`
	Branch     string `json:"branch,omitempty"`
	Image      string `json:"image,omitempty"`
	SetupCmd   string `json:"setupCmd,omitempty"`
	Command    string `json:"command"`
	ReportPath string `json:"reportPath"`
	// ArtifactsPath is a directory the suite writes recordings and screenshots
	// into; empty means the suite produces none.
	ArtifactsPath string            `json:"artifactsPath"`
	EnvVars       map[string]string `json:"envVars"`
	TimeoutMs     int               `json:"timeoutMs"`
	CreatedAt     time.Time         `json:"$createdAt"`
	UpdatedAt     time.Time         `json:"$updatedAt"`
}

// Run is one execution of a suite.
type Run struct {
	ID           string     `json:"$id"`
	ProjectID    string     `json:"projectId"`
	SuiteID      string     `json:"suiteId"`
	Status       string     `json:"status"`
	Target       string     `json:"target"`
	TriggerType  string     `json:"triggerType"`
	TriggerActor string     `json:"triggerActor,omitempty"`
	CommitSHA    string     `json:"commitSha,omitempty"`
	Total        int        `json:"total"`
	Passed       int        `json:"passed"`
	Failed       int        `json:"failed"`
	Skipped      int        `json:"skipped"`
	DurationMs   int64      `json:"durationMs"`
	Log          string     `json:"log,omitempty"`
	Error        string     `json:"error,omitempty"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
	CreatedAt    time.Time  `json:"$createdAt"`
}

// CaseResult is a stored test case, as returned by the API.
type CaseResult struct {
	ID             string    `json:"$id"`
	RunID          string    `json:"runId"`
	SuiteName      string    `json:"suiteName"`
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	DurationMs     int64     `json:"durationMs"`
	FailureMessage string    `json:"failureMessage,omitempty"`
	FailureDetails string    `json:"failureDetails,omitempty"`
	SpecRef        string    `json:"specRef,omitempty"`
	CreatedAt      time.Time `json:"$createdAt"`
}

// ── Suites ──

func (s *Service) CreateSuite(ctx context.Context, projectID string, in Suite) (*Suite, error) {
	if in.ReportPath == "" {
		in.ReportPath = "junit.xml"
	}
	if in.TimeoutMs <= 0 {
		in.TimeoutMs = 900000
	}
	if in.SourceType == "" {
		in.SourceType = "upload"
	}
	if in.EnvVars == nil {
		in.EnvVars = map[string]string{}
	}
	envJSON, _ := json.Marshal(in.EnvVars)

	id := uid.New("unique()")
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO test_suites (id, project_id, name, source_type, source_url, branch, image,
		                          setup_cmd, command, report_path, artifacts_path, env_vars, timeout_ms, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		id, projectID, in.Name, in.SourceType, in.SourceURL, in.Branch, in.Image,
		in.SetupCmd, in.Command, in.ReportPath, in.ArtifactsPath, envJSON, in.TimeoutMs, now, now)
	if err != nil {
		return nil, fmt.Errorf("testlab: create suite: %w", err)
	}

	in.ID, in.ProjectID, in.CreatedAt, in.UpdatedAt = id, projectID, now, now
	return &in, nil
}

func (s *Service) UpdateSuite(ctx context.Context, id, projectID string, in Suite) (*Suite, error) {
	if in.ReportPath == "" {
		in.ReportPath = "junit.xml"
	}
	if in.TimeoutMs <= 0 {
		in.TimeoutMs = 900000
	}
	if in.EnvVars == nil {
		in.EnvVars = map[string]string{}
	}
	envJSON, _ := json.Marshal(in.EnvVars)

	_, err := s.db.ExecContext(ctx,
		`UPDATE test_suites SET name=$1, source_type=$2, source_url=$3, branch=$4, image=$5,
		        setup_cmd=$6, command=$7, report_path=$8, artifacts_path=$9, env_vars=$10, timeout_ms=$11
		  WHERE id=$12 AND project_id=$13`,
		in.Name, in.SourceType, in.SourceURL, in.Branch, in.Image,
		in.SetupCmd, in.Command, in.ReportPath, in.ArtifactsPath, envJSON, in.TimeoutMs, id, projectID)
	if err != nil {
		return nil, fmt.Errorf("testlab: update suite: %w", err)
	}
	return s.GetSuite(ctx, id, projectID)
}

func (s *Service) GetSuite(ctx context.Context, id, projectID string) (*Suite, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, source_type, COALESCE(source_url,''), COALESCE(branch,''),
		        COALESCE(image,''), COALESCE(setup_cmd,''), command, report_path, artifacts_path,
		        COALESCE(env_vars,'{}'), timeout_ms, created_at, updated_at
		   FROM test_suites WHERE id = $1 AND project_id = $2`, id, projectID)
	return scanSuite(row)
}

func (s *Service) ListSuites(ctx context.Context, projectID string) ([]*Suite, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, source_type, COALESCE(source_url,''), COALESCE(branch,''),
		        COALESCE(image,''), COALESCE(setup_cmd,''), command, report_path, artifacts_path,
		        COALESCE(env_vars,'{}'), timeout_ms, created_at, updated_at
		   FROM test_suites WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, 0, fmt.Errorf("testlab: list suites: %w", err)
	}
	defer rows.Close()

	var out []*Suite
	for rows.Next() {
		suite, err := scanSuite(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, suite)
	}
	return out, len(out), nil
}

func (s *Service) DeleteSuite(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM test_suites WHERE id = $1 AND project_id = $2", id, projectID)
	return err
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanSuite(row scanner) (*Suite, error) {
	var s Suite
	var envJSON []byte
	if err := row.Scan(&s.ID, &s.ProjectID, &s.Name, &s.SourceType, &s.SourceURL, &s.Branch,
		&s.Image, &s.SetupCmd, &s.Command, &s.ReportPath, &s.ArtifactsPath, &envJSON, &s.TimeoutMs,
		&s.CreatedAt, &s.UpdatedAt); err != nil {
		return nil, err
	}
	s.EnvVars = map[string]string{}
	json.Unmarshal(envJSON, &s.EnvVars) //nolint:errcheck
	return &s, nil
}

// ── Runs ──

// Trigger queues a run of the suite. The work happens on the builds worker,
// which has the Docker socket.
func (s *Service) Trigger(ctx context.Context, suiteID, projectID, triggerType, actor string) (*Run, error) {
	// Confirms the suite exists and belongs to this project before a run row
	// is created for it.
	if _, err := s.GetSuite(ctx, suiteID, projectID); err != nil {
		return nil, fmt.Errorf("testlab: suite not found")
	}

	id := uid.New("unique()")
	now := time.Now().UTC()
	if triggerType == "" {
		triggerType = "manual"
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO test_runs (id, project_id, suite_id, status, target, trigger_type, trigger_actor, created_at)
		 VALUES ($1,$2,$3,'queued','container',$4,$5,$6)`,
		id, projectID, suiteID, triggerType, actor, now)
	if err != nil {
		return nil, fmt.Errorf("testlab: create run: %w", err)
	}

	if s.queue != nil {
		s.queue.Push(ctx, "builds", queue.Job{ //nolint:errcheck
			ID:   id,
			Type: "test_run",
			Payload: map[string]interface{}{
				"runId": id, "suiteId": suiteID, "projectId": projectID,
			},
			CreatedAt: now,
		})
	}

	return &Run{
		ID: id, ProjectID: projectID, SuiteID: suiteID, Status: "queued",
		Target: "container", TriggerType: triggerType, TriggerActor: actor,
		CreatedAt: now,
	}, nil
}

func (s *Service) GetRun(ctx context.Context, id, projectID string) (*Run, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, suite_id, status, target, trigger_type, COALESCE(trigger_actor,''),
		        COALESCE(commit_sha,''), total, passed, failed, skipped, duration_ms,
		        COALESCE(log,''), COALESCE(error,''), started_at, finished_at, created_at
		   FROM test_runs WHERE id = $1 AND project_id = $2`, id, projectID)
	return scanRun(row)
}

func (s *Service) ListRuns(ctx context.Context, projectID, suiteID string, limit int) ([]*Run, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `SELECT id, project_id, suite_id, status, target, trigger_type, COALESCE(trigger_actor,''),
	                 COALESCE(commit_sha,''), total, passed, failed, skipped, duration_ms,
	                 '', COALESCE(error,''), started_at, finished_at, created_at
	            FROM test_runs WHERE project_id = $1`
	args := []interface{}{projectID}
	if suiteID != "" {
		query += " AND suite_id = $2"
		args = append(args, suiteID)
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d", limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("testlab: list runs: %w", err)
	}
	defer rows.Close()

	var out []*Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, len(out), nil
}

func scanRun(row scanner) (*Run, error) {
	var r Run
	var started, finished sql.NullTime
	if err := row.Scan(&r.ID, &r.ProjectID, &r.SuiteID, &r.Status, &r.Target, &r.TriggerType,
		&r.TriggerActor, &r.CommitSHA, &r.Total, &r.Passed, &r.Failed, &r.Skipped,
		&r.DurationMs, &r.Log, &r.Error, &started, &finished, &r.CreatedAt); err != nil {
		return nil, err
	}
	if started.Valid {
		r.StartedAt = &started.Time
	}
	if finished.Valid {
		r.FinishedAt = &finished.Time
	}
	return &r, nil
}

// ListCases returns the individual results of a run, failures first: a red run
// is read by looking at what broke, not by scrolling past what passed.
func (s *Service) ListCases(ctx context.Context, runID, projectID string) ([]*CaseResult, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.id, c.run_id, c.suite_name, c.name, c.status, c.duration_ms,
		        COALESCE(c.failure_message,''), COALESCE(c.failure_details,''), COALESCE(c.spec_ref,''), c.created_at
		   FROM test_cases c
		   JOIN test_runs r ON r.id = c.run_id
		  WHERE c.run_id = $1 AND r.project_id = $2
		  ORDER BY CASE c.status WHEN 'errored' THEN 0 WHEN 'failed' THEN 1 WHEN 'skipped' THEN 2 ELSE 3 END,
		           c.suite_name, c.name`, runID, projectID)
	if err != nil {
		return nil, 0, fmt.Errorf("testlab: list cases: %w", err)
	}
	defer rows.Close()

	var out []*CaseResult
	for rows.Next() {
		var c CaseResult
		if err := rows.Scan(&c.ID, &c.RunID, &c.SuiteName, &c.Name, &c.Status, &c.DurationMs,
			&c.FailureMessage, &c.FailureDetails, &c.SpecRef, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, &c)
	}
	return out, len(out), nil
}

// RecordResults stores a finished run's cases and its verdict, in one
// transaction so a run is never half-recorded.
func (s *Service) RecordResults(ctx context.Context, runID, projectID string, cases []Case, log, runErr string, durationMs int64) error {
	summary := Summarise(cases)

	status := "passed"
	switch {
	case runErr != "":
		status = "errored"
	case summary.Failed > 0:
		status = "failed"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("testlab: record results: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx,
		`UPDATE test_runs SET status=$1, total=$2, passed=$3, failed=$4, skipped=$5,
		        duration_ms=$6, log=$7, error=NULLIF($8,''), finished_at=$9
		  WHERE id=$10 AND project_id=$11`,
		status, summary.Total, summary.Passed, summary.Failed, summary.Skipped,
		durationMs, log, runErr, now, runID, projectID); err != nil {
		return fmt.Errorf("testlab: record results: %w", err)
	}

	for _, c := range cases {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO test_cases (id, run_id, project_id, suite_name, name, status,
			                         duration_ms, failure_message, failure_details, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),NULLIF($9,''),$10)`,
			uid.New("unique()"), runID, projectID, c.SuiteName, c.Name, string(c.Status),
			c.DurationMs, c.FailureMessage, c.FailureDetails, now); err != nil {
			return fmt.Errorf("testlab: record case: %w", err)
		}
	}

	return tx.Commit()
}

// MarkRunning records that a run has started.
func (s *Service) MarkRunning(ctx context.Context, runID string) {
	s.db.ExecContext(ctx, //nolint:errcheck
		"UPDATE test_runs SET status='running', started_at=$1 WHERE id=$2", time.Now().UTC(), runID)
}

// ── Flows ──

// SaveFlow stores a recording and creates the suite that runs it.
//
// The generated project is a complete Playwright package, so what gets saved
// is something the existing runner can execute unchanged and a person can read
// and edit. A recording that cannot be run is a demo, not a test.
func (s *Service) SaveFlow(ctx context.Context, projectID string, f Flow) (*Flow, error) {
	f.ID = uid.New("unique()")
	f.ProjectID = projectID
	if f.Platform == "" {
		f.Platform = "web"
	}

	spec := CompilePlaywright(f)
	suite, err := s.CreateSuite(ctx, projectID, Suite{
		Name:          f.Name,
		SourceType:    "generated",
		Image:         browserTestImage(),
		SetupCmd:      "npm install --no-audit --no-fund @playwright/test@1.49.0",
		Command:       "npx playwright test",
		ReportPath:    "junit.xml",
		ArtifactsPath: "test-results",
		EnvVars:       map[string]string{"BASE_URL": f.Target},
		TimeoutMs:     900000,
	})
	if err != nil {
		return nil, err
	}
	f.SuiteID = suite.ID

	// The suite's source is the recording, written where the runner expects an
	// uploaded project.
	if err := writeGeneratedProject(suite.ID, f.Name, spec); err != nil {
		return nil, fmt.Errorf("testlab: write generated suite: %w", err)
	}

	stepsJSON, _ := json.Marshal(f.Steps)
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO test_flows (id, project_id, name, platform, target, steps, suite_id, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8)`,
		f.ID, projectID, f.Name, f.Platform, f.Target, stepsJSON, f.SuiteID, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("testlab: save flow: %w", err)
	}
	return &f, nil
}

func (s *Service) ListFlows(ctx context.Context, projectID string) ([]*Flow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, platform, target, steps, COALESCE(suite_id,'')
		   FROM test_flows WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("testlab: list flows: %w", err)
	}
	defer rows.Close()

	var out []*Flow
	for rows.Next() {
		f, err := scanFlow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func (s *Service) GetFlow(ctx context.Context, id, projectID string) (*Flow, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, platform, target, steps, COALESCE(suite_id,'')
		   FROM test_flows WHERE id = $1 AND project_id = $2`, id, projectID)
	return scanFlow(row)
}

func (s *Service) DeleteFlow(ctx context.Context, id, projectID string) error {
	// The suite generated from the flow goes with it: keeping a test whose
	// recording was discarded leaves something nobody can explain.
	var suiteID string
	s.db.QueryRowContext(ctx, //nolint:errcheck
		"SELECT COALESCE(suite_id,'') FROM test_flows WHERE id = $1 AND project_id = $2",
		id, projectID).Scan(&suiteID)

	if _, err := s.db.ExecContext(ctx,
		"DELETE FROM test_flows WHERE id = $1 AND project_id = $2", id, projectID); err != nil {
		return err
	}
	if suiteID != "" {
		return s.DeleteSuite(ctx, suiteID, projectID)
	}
	return nil
}

func scanFlow(row scanner) (*Flow, error) {
	var f Flow
	var stepsJSON []byte
	if err := row.Scan(&f.ID, &f.ProjectID, &f.Name, &f.Platform, &f.Target, &stepsJSON, &f.SuiteID); err != nil {
		return nil, err
	}
	json.Unmarshal(stepsJSON, &f.Steps) //nolint:errcheck
	return &f, nil
}
