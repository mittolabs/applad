package testlab

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

// Runner is how a body of tests is executed: the image, the commands, and
// where results and evidence land. A project usually has one or two.
type Runner struct {
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
	RunnerID     string     `json:"runnerId"`
	SuiteID      string     `json:"suiteId,omitempty"`
	TargetURL    string     `json:"targetUrl,omitempty"`
	Flaky        int        `json:"flaky"`
	Quarantined  int        `json:"quarantined"`
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

func (s *Service) CreateRunner(ctx context.Context, projectID string, in Runner) (*Runner, error) {
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
		`INSERT INTO test_runners (id, project_id, name, source_type, source_url, branch, image,
		                          setup_cmd, command, report_path, artifacts_path, env_vars, timeout_ms, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		id, projectID, in.Name, in.SourceType, in.SourceURL, in.Branch, in.Image,
		in.SetupCmd, in.Command, in.ReportPath, in.ArtifactsPath, envJSON, in.TimeoutMs, now, now)
	if err != nil {
		return nil, fmt.Errorf("testlab: create runner: %w", err)
	}

	in.ID, in.ProjectID, in.CreatedAt, in.UpdatedAt = id, projectID, now, now
	return &in, nil
}

func (s *Service) UpdateRunner(ctx context.Context, id, projectID string, in Runner) (*Runner, error) {
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
		`UPDATE test_runners SET name=$1, source_type=$2, source_url=$3, branch=$4, image=$5,
		        setup_cmd=$6, command=$7, report_path=$8, artifacts_path=$9, env_vars=$10, timeout_ms=$11
		  WHERE id=$12 AND project_id=$13`,
		in.Name, in.SourceType, in.SourceURL, in.Branch, in.Image,
		in.SetupCmd, in.Command, in.ReportPath, in.ArtifactsPath, envJSON, in.TimeoutMs, id, projectID)
	if err != nil {
		return nil, fmt.Errorf("testlab: update runner: %w", err)
	}
	return s.GetRunner(ctx, id, projectID)
}

func (s *Service) GetRunner(ctx context.Context, id, projectID string) (*Runner, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, source_type, COALESCE(source_url,''), COALESCE(branch,''),
		        COALESCE(image,''), COALESCE(setup_cmd,''), command, report_path, artifacts_path,
		        COALESCE(env_vars,'{}'), timeout_ms, created_at, updated_at
		   FROM test_runners WHERE id = $1 AND project_id = $2`, id, projectID)
	return scanRunner(row)
}

func (s *Service) ListRunners(ctx context.Context, projectID string) ([]*Runner, int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, source_type, COALESCE(source_url,''), COALESCE(branch,''),
		        COALESCE(image,''), COALESCE(setup_cmd,''), command, report_path, artifacts_path,
		        COALESCE(env_vars,'{}'), timeout_ms, created_at, updated_at
		   FROM test_runners WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, 0, fmt.Errorf("testlab: list runners: %w", err)
	}
	defer rows.Close()

	var out []*Runner
	for rows.Next() {
		runner, err := scanRunner(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, runner)
	}
	return out, len(out), nil
}

func (s *Service) DeleteRunner(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM test_runners WHERE id = $1 AND project_id = $2", id, projectID)
	return err
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanRunner(row scanner) (*Runner, error) {
	var s Runner
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
func (s *Service) Trigger(ctx context.Context, runnerID, projectID, triggerType, actor string, opts TriggerOptions) (*Run, error) {
	runner, err := s.GetRunner(ctx, runnerID, projectID)
	if err != nil {
		return nil, fmt.Errorf("testlab: runner not found")
	}

	// A selection narrows what runs; a target says what it runs against. Both
	// belong to the run rather than the runner, which is what lets one suite
	// be pointed at main and at a branch.
	var selection *Selection
	if opts.SuiteID != "" {
		selection, err = s.GetSelection(ctx, opts.SuiteID, projectID)
		if err != nil {
			return nil, fmt.Errorf("testlab: suite not found")
		}
		if opts.Target == "" {
			opts.Target = selection.DefaultTarget
		}
	}
	if opts.Target == "" {
		opts.Target = runner.EnvVars["BASE_URL"]
	}

	// The flows are the source of truth and the generated project is derived
	// from them, so it is rebuilt before it runs rather than being kept in
	// step by every path that could change a recording.
	if runner.SourceType == "generated" {
		if err := s.regenerateRecordedProject(ctx, projectID, runner.ID); err != nil {
			return nil, err
		}
	}

	names, err := s.SelectedNames(ctx, projectID, selection)
	if err != nil {
		return nil, err
	}

	id := uid.New("unique()")
	now := time.Now().UTC()
	if triggerType == "" {
		triggerType = "manual"
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO test_runs (id, project_id, runner_id, suite_id, status, target, target_url,
		                        trigger_type, trigger_actor, created_at)
		 VALUES ($1,$2,$3,NULLIF($4,''),'queued','container',$5,$6,$7,$8)`,
		id, projectID, runnerID, opts.SuiteID, opts.Target, triggerType, actor, now)
	if err != nil {
		return nil, fmt.Errorf("testlab: create run: %w", err)
	}

	if s.queue != nil {
		s.queue.Push(ctx, "builds", queue.Job{ //nolint:errcheck
			ID:   id,
			Type: "test_run",
			Payload: map[string]interface{}{
				"runId": id, "runnerId": runnerID, "projectId": projectID,
				"target": opts.Target, "grep": grepFor(names),
			},
			CreatedAt: now,
		})
	}

	return &Run{
		ID: id, ProjectID: projectID, RunnerID: runnerID, SuiteID: opts.SuiteID,
		Status: "queued", Target: "container", TargetURL: opts.Target,
		TriggerType: triggerType, TriggerActor: actor, CreatedAt: now,
	}, nil
}

// TriggerOptions are the things that vary per run rather than per runner.
type TriggerOptions struct {
	// SuiteID narrows the run to a selection.
	SuiteID string
	// Target overrides what the tests run against — a branch rather than main.
	Target string
}

func (s *Service) GetRun(ctx context.Context, id, projectID string) (*Run, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, runner_id, COALESCE(suite_id,''), status, target, COALESCE(target_url,''),
		        trigger_type, COALESCE(trigger_actor,''), COALESCE(commit_sha,''),
		        total, passed, failed, skipped, flaky, quarantined, duration_ms,
		        COALESCE(log,''), COALESCE(error,''), started_at, finished_at, created_at
		   FROM test_runs WHERE id = $1 AND project_id = $2`, id, projectID)
	return scanRun(row)
}

func (s *Service) ListRuns(ctx context.Context, projectID, runnerID string, limit int) ([]*Run, int, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `SELECT id, project_id, runner_id, COALESCE(suite_id,''), status, target, COALESCE(target_url,''),
	                 trigger_type, COALESCE(trigger_actor,''), COALESCE(commit_sha,''),
	                 total, passed, failed, skipped, flaky, quarantined, duration_ms,
	                 '', COALESCE(error,''), started_at, finished_at, created_at
	            FROM test_runs WHERE project_id = $1`
	args := []interface{}{projectID}
	if runnerID != "" {
		query += " AND runner_id = $2"
		args = append(args, runnerID)
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
	if err := row.Scan(&r.ID, &r.ProjectID, &r.RunnerID, &r.SuiteID, &r.Status, &r.Target, &r.TargetURL,
		&r.TriggerType, &r.TriggerActor, &r.CommitSHA, &r.Total, &r.Passed, &r.Failed, &r.Skipped,
		&r.Flaky, &r.Quarantined, &r.DurationMs, &r.Log, &r.Error, &started, &finished, &r.CreatedAt); err != nil {
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
func (s *Service) RecordResults(ctx context.Context, runID, projectID, runnerID string, cases []Case, log, runErr string, durationMs int64) error {
	now := time.Now().UTC()

	// Discovery: every test a run reports becomes a catalogue entry, which is
	// how authored tests gain a history without anybody registering them.
	testIDs, err := s.RecordDiscovered(ctx, projectID, runnerID, cases, now)
	if err != nil {
		return err
	}
	quarantined, err := s.QuarantinedNames(ctx, projectID)
	if err != nil {
		return err
	}

	summary := Summarise(cases)
	blocking, flaky, quarantinedCount := 0, 0, 0
	for _, c := range cases {
		if c.Flaky {
			flaky++
		}
		if c.Status == CasePassed || c.Status == CaseSkipped {
			continue
		}
		// A quarantined failure is reported but does not decide the run: that
		// is the whole point of quarantining rather than deleting.
		if quarantined[c.SuiteName+"\x00"+c.Name] {
			quarantinedCount++
			continue
		}
		blocking++
	}

	status := "passed"
	switch {
	case runErr != "":
		status = "errored"
	case blocking > 0:
		status = "failed"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("testlab: record results: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx,
		`UPDATE test_runs SET status=$1, total=$2, passed=$3, failed=$4, skipped=$5,
		        flaky=$6, quarantined=$7, duration_ms=$8, log=$9, error=NULLIF($10,''), finished_at=$11
		  WHERE id=$12 AND project_id=$13`,
		status, summary.Total, summary.Passed, summary.Failed, summary.Skipped,
		flaky, quarantinedCount, durationMs, log, runErr, now, runID, projectID); err != nil {
		return fmt.Errorf("testlab: record results: %w", err)
	}

	for _, c := range cases {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO test_cases (id, run_id, project_id, test_id, suite_name, name, status,
			                         duration_ms, failure_message, failure_details, flaky, retries, created_at)
			 VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8,NULLIF($9,''),NULLIF($10,''),$11,$12,$13)`,
			uid.New("unique()"), runID, projectID, testIDs[c.SuiteName+"\x00"+c.Name],
			c.SuiteName, c.Name, string(c.Status), c.DurationMs,
			c.FailureMessage, c.FailureDetails, c.Flaky, c.Retries, now); err != nil {
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

	// Every recording shares one runner. Giving each its own is what made a
	// recording appear twice — once as a flow, once as a suite — and made
	// "suite" mean "a single test".
	runner, err := s.RecordedRunner(ctx, projectID)
	if err != nil {
		return nil, err
	}
	f.RunnerID = runner.ID

	stepsJSON, _ := json.Marshal(f.Steps)
	now := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO test_flows (id, project_id, name, platform, target, steps, runner_id, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8)`,
		f.ID, projectID, f.Name, f.Platform, f.Target, stepsJSON, runner.ID, now); err != nil {
		return nil, fmt.Errorf("testlab: save flow: %w", err)
	}

	// A recorded test is known before it ever runs, so it joins the catalogue
	// immediately rather than waiting to be discovered.
	var testID string
	if err := s.db.QueryRowContext(ctx,
		`INSERT INTO tests (id, project_id, runner_id, suite_name, name, source, flow_id, created_at, updated_at)
		 VALUES ($1,$2,$3,'recorded',$4,'recorded',$5,$6,$6)
		 ON CONFLICT (project_id, runner_id, suite_name, name) DO UPDATE SET flow_id = EXCLUDED.flow_id
		 RETURNING id`,
		uid.New("unique()"), projectID, runner.ID, f.Name, f.ID, now).Scan(&testID); err != nil {
		return nil, fmt.Errorf("testlab: catalogue recorded test: %w", err)
	}
	s.db.ExecContext(ctx, "UPDATE test_flows SET test_id = $1 WHERE id = $2", testID, f.ID) //nolint:errcheck

	// The generated project holds every recording, so a selection can run one
	// of them or all of them.
	if err := s.regenerateRecordedProject(ctx, projectID, runner.ID); err != nil {
		return nil, err
	}
	return &f, nil
}

// RecordedRunner returns the single runner recordings compile into, creating
// it the first time something is recorded.
func (s *Service) RecordedRunner(ctx context.Context, projectID string) (*Runner, error) {
	var id string
	err := s.db.QueryRowContext(ctx,
		"SELECT id FROM test_runners WHERE project_id = $1 AND source_type = 'generated' LIMIT 1",
		projectID).Scan(&id)
	if err == nil {
		return s.GetRunner(ctx, id, projectID)
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("testlab: find recorded runner: %w", err)
	}

	return s.CreateRunner(ctx, projectID, Runner{
		Name:          "Recorded flows",
		SourceType:    "generated",
		Image:         browserTestImage(),
		SetupCmd:      "npm install --no-audit --no-fund @playwright/test@1.49.0",
		Command:       "npx playwright test",
		ReportPath:    "junit.xml",
		ArtifactsPath: "test-results",
		TimeoutMs:     900000,
	})
}

// regenerateRecordedProject rewrites the generated project from every flow, so
// adding or discarding one keeps the runnable source in step with the
// catalogue.
func (s *Service) regenerateRecordedProject(ctx context.Context, projectID, runnerID string) error {
	flows, err := s.ListFlows(ctx, projectID)
	if err != nil {
		return err
	}
	specs := map[string]string{}
	for _, f := range flows {
		specs[specFileName(f.Name)] = CompilePlaywright(*f)
	}
	if err := writeGeneratedProject(runnerID, specs); err != nil {
		return fmt.Errorf("testlab: write generated project: %w", err)
	}
	return nil
}

// specFileName turns a flow's name into a file name that stays readable in a
// stack trace.
func specFileName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-") + ".spec.js"
}

func (s *Service) ListFlows(ctx context.Context, projectID string) ([]*Flow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, platform, target, steps, COALESCE(runner_id,'')
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
		`SELECT id, project_id, name, platform, target, steps, COALESCE(runner_id,'')
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
		return s.DeleteRunner(ctx, suiteID, projectID)
	}
	return nil
}

func scanFlow(row scanner) (*Flow, error) {
	var f Flow
	var stepsJSON []byte
	if err := row.Scan(&f.ID, &f.ProjectID, &f.Name, &f.Platform, &f.Target, &stepsJSON, &f.RunnerID); err != nil {
		return nil, err
	}
	json.Unmarshal(stepsJSON, &f.Steps) //nolint:errcheck
	return &f, nil
}
