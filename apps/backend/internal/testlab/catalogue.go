package testlab

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/uid"
)

/*
 * The test catalogue, and the selections over it.
 *
 * A test is one checkable behaviour that outlives any particular run. Recorded
 * flows create one directly; authored tests are discovered the first time a
 * run reports them, since that is the only moment their names exist. Either
 * way the entry is stable, which is what makes "when did this start failing"
 * and "is this flaky" answerable at all.
 */

// Test is one behaviour in the catalogue.
type Test struct {
	ID          string     `json:"$id"`
	ProjectID   string     `json:"projectId"`
	RunnerID    string     `json:"runnerId"`
	SuiteName   string     `json:"suiteName"`
	Name        string     `json:"name"`
	Source      string     `json:"source"`
	FlowID      string     `json:"flowId,omitempty"`
	Tags        []string   `json:"tags"`
	Quarantined bool       `json:"quarantined"`
	LastStatus  string     `json:"lastStatus,omitempty"`
	LastRunAt   *time.Time `json:"lastRunAt,omitempty"`
	// History is the recent record, newest first, for a trend at a glance.
	History []string `json:"history,omitempty"`
}

// Selection is a named set of tests and when it should run.
type Selection struct {
	ID            string   `json:"$id"`
	ProjectID     string   `json:"projectId"`
	Name          string   `json:"name"`
	Tags          []string `json:"tags"`
	RunnerID      string   `json:"runnerId,omitempty"`
	DefaultTarget string   `json:"defaultTarget"`
	RunOnDeploy   bool     `json:"runOnDeploy"`
	Cron          string   `json:"cron,omitempty"`
	// TestCount is filled on read so a selection can be judged without opening it.
	TestCount int `json:"testCount"`
}

// ── Catalogue ──

// RecordDiscovered upserts the tests a run reported, so authored tests appear
// in the catalogue without anybody registering them.
func (s *Service) RecordDiscovered(ctx context.Context, projectID, runnerID string, cases []Case, at time.Time) (map[string]string, error) {
	ids := make(map[string]string, len(cases))
	for _, c := range cases {
		var id string
		err := s.db.QueryRowContext(ctx,
			`INSERT INTO tests (id, project_id, runner_id, suite_name, name, source, last_status, last_run_at, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,'discovered',$6,$7,$8,$8)
			 ON CONFLICT (project_id, runner_id, suite_name, name) DO UPDATE
			   SET last_status = EXCLUDED.last_status, last_run_at = EXCLUDED.last_run_at
			 RETURNING id`,
			uid.New("unique()"), projectID, runnerID, c.SuiteName, c.Name, string(c.Status), at, at).Scan(&id)
		if err != nil {
			return ids, fmt.Errorf("testlab: record test: %w", err)
		}
		ids[c.SuiteName+"\x00"+c.Name] = id
	}
	return ids, nil
}

// ListTests returns the catalogue with a short history for each entry.
func (s *Service) ListTests(ctx context.Context, projectID string) ([]*Test, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT t.id, t.project_id, t.runner_id, t.suite_name, t.name, t.source,
		        COALESCE(t.flow_id,''), t.tags, t.quarantined,
		        COALESCE(t.last_status,''), t.last_run_at
		   FROM tests t WHERE t.project_id = $1
		  ORDER BY t.suite_name, t.name`, projectID)
	if err != nil {
		return nil, fmt.Errorf("testlab: list tests: %w", err)
	}
	defer rows.Close()

	var out []*Test
	for rows.Next() {
		var t Test
		var tagsJSON []byte
		var lastRun sql.NullTime
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.RunnerID, &t.SuiteName, &t.Name, &t.Source,
			&t.FlowID, &tagsJSON, &t.Quarantined, &t.LastStatus, &lastRun); err != nil {
			return nil, err
		}
		json.Unmarshal(tagsJSON, &t.Tags) //nolint:errcheck
		if t.Tags == nil {
			t.Tags = []string{}
		}
		if lastRun.Valid {
			t.LastRunAt = &lastRun.Time
		}
		out = append(out, &t)
	}

	// One query for every test's recent history rather than one per test.
	if len(out) > 0 {
		if err := s.attachHistory(ctx, projectID, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Service) attachHistory(ctx context.Context, projectID string, tests []*Test) error {
	rows, err := s.db.QueryContext(ctx,
		`SELECT test_id, status FROM (
		   SELECT test_id, status, created_at,
		          ROW_NUMBER() OVER (PARTITION BY test_id ORDER BY created_at DESC) AS rn
		     FROM test_cases WHERE project_id = $1 AND test_id IS NOT NULL
		 ) ranked WHERE rn <= 10 ORDER BY test_id, created_at DESC`, projectID)
	if err != nil {
		return fmt.Errorf("testlab: history: %w", err)
	}
	defer rows.Close()

	history := map[string][]string{}
	for rows.Next() {
		var id, status string
		if err := rows.Scan(&id, &status); err != nil {
			return err
		}
		history[id] = append(history[id], status)
	}
	for _, t := range tests {
		t.History = history[t.ID]
	}
	return nil
}

// SetTags replaces a test's tags, which is how selections are built.
func (s *Service) SetTags(ctx context.Context, testID, projectID string, tags []string) error {
	if tags == nil {
		tags = []string{}
	}
	data, _ := json.Marshal(tags)
	_, err := s.db.ExecContext(ctx,
		"UPDATE tests SET tags = $1 WHERE id = $2 AND project_id = $3", data, testID, projectID)
	return err
}

// SetQuarantined takes a test out of a run's verdict without deleting it.
func (s *Service) SetQuarantined(ctx context.Context, testID, projectID string, on bool) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE tests SET quarantined = $1 WHERE id = $2 AND project_id = $3", on, testID, projectID)
	return err
}

// ── Selections ──

func (s *Service) CreateSelection(ctx context.Context, projectID string, in Selection) (*Selection, error) {
	if in.Tags == nil {
		in.Tags = []string{}
	}
	tags, _ := json.Marshal(in.Tags)
	in.ID = uid.New("unique()")
	in.ProjectID = projectID
	now := time.Now().UTC()

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO test_suites (id, project_id, name, tags, runner_id, default_target, run_on_deploy, cron, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7,NULLIF($8,''),$9,$9)`,
		in.ID, projectID, in.Name, tags, in.RunnerID, in.DefaultTarget, in.RunOnDeploy, in.Cron, now)
	if err != nil {
		return nil, fmt.Errorf("testlab: create suite: %w", err)
	}
	return &in, nil
}

func (s *Service) UpdateSelection(ctx context.Context, id, projectID string, in Selection) (*Selection, error) {
	if in.Tags == nil {
		in.Tags = []string{}
	}
	tags, _ := json.Marshal(in.Tags)
	_, err := s.db.ExecContext(ctx,
		`UPDATE test_suites SET name=$1, tags=$2, runner_id=NULLIF($3,''), default_target=$4,
		        run_on_deploy=$5, cron=NULLIF($6,'')
		  WHERE id=$7 AND project_id=$8`,
		in.Name, tags, in.RunnerID, in.DefaultTarget, in.RunOnDeploy, in.Cron, id, projectID)
	if err != nil {
		return nil, fmt.Errorf("testlab: update suite: %w", err)
	}
	return s.GetSelection(ctx, id, projectID)
}

func (s *Service) GetSelection(ctx context.Context, id, projectID string) (*Selection, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, tags, COALESCE(runner_id,''), default_target,
		        run_on_deploy, COALESCE(cron,'')
		   FROM test_suites WHERE id = $1 AND project_id = $2`, id, projectID)
	return scanSelection(row)
}

func (s *Service) ListSelections(ctx context.Context, projectID string) ([]*Selection, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, tags, COALESCE(runner_id,''), default_target,
		        run_on_deploy, COALESCE(cron,'')
		   FROM test_suites WHERE project_id = $1 ORDER BY name`, projectID)
	if err != nil {
		return nil, fmt.Errorf("testlab: list suites: %w", err)
	}
	defer rows.Close()

	var out []*Selection
	for rows.Next() {
		sel, err := scanSelection(rows)
		if err != nil {
			return nil, err
		}
		if n, err := s.countSelected(ctx, projectID, sel); err == nil {
			sel.TestCount = n
		}
		out = append(out, sel)
	}
	return out, nil
}

func (s *Service) DeleteSelection(ctx context.Context, id, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM test_suites WHERE id = $1 AND project_id = $2", id, projectID)
	return err
}

func scanSelection(row scanner) (*Selection, error) {
	var sel Selection
	var tagsJSON []byte
	if err := row.Scan(&sel.ID, &sel.ProjectID, &sel.Name, &tagsJSON, &sel.RunnerID,
		&sel.DefaultTarget, &sel.RunOnDeploy, &sel.Cron); err != nil {
		return nil, err
	}
	json.Unmarshal(tagsJSON, &sel.Tags) //nolint:errcheck
	if sel.Tags == nil {
		sel.Tags = []string{}
	}
	return &sel, nil
}

func (s *Service) countSelected(ctx context.Context, projectID string, sel *Selection) (int, error) {
	query := "SELECT COUNT(*) FROM tests WHERE project_id = $1"
	args := []interface{}{projectID}
	if sel.RunnerID != "" {
		query += " AND runner_id = $2"
		args = append(args, sel.RunnerID)
	}
	if len(sel.Tags) > 0 {
		// Written without the ?| operator: the driver rewrites ? as a
		// placeholder, so a JSONB operator containing one silently breaks.
		query += fmt.Sprintf(
			" AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(tags) tag"+
				" WHERE tag = ANY(string_to_array($%d, ',')))", len(args)+1)
		args = append(args, strings.Join(sel.Tags, ","))
	}
	var n int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&n)
	return n, err
}

// SelectedNames returns the test names a selection covers, which the runner
// uses to run only those rather than everything.
func (s *Service) SelectedNames(ctx context.Context, projectID string, sel *Selection) ([]string, error) {
	if sel == nil || len(sel.Tags) == 0 {
		return nil, nil // everything
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT name FROM tests
		  WHERE project_id = $1
		    AND EXISTS (SELECT 1 FROM jsonb_array_elements_text(tags) tag
		                 WHERE tag = ANY(string_to_array($2, ',')))`,
		projectID, strings.Join(sel.Tags, ","))
	if err != nil {
		return nil, fmt.Errorf("testlab: selected names: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, nil
}

// QuarantinedNames returns tests excluded from a run's verdict.
func (s *Service) QuarantinedNames(ctx context.Context, projectID string) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT suite_name, name FROM tests WHERE project_id = $1 AND quarantined", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var suite, name string
		if err := rows.Scan(&suite, &name); err != nil {
			return nil, err
		}
		out[suite+"\x00"+name] = true
	}
	return out, nil
}

// grepFor renders a selection as a test-name filter the runner understands.
// Playwright, Jest, pytest and go test all accept one; passing names rather
// than rewriting the project keeps the runner ignorant of selections.
func grepFor(names []string) string {
	if len(names) == 0 {
		return ""
	}
	escaped := make([]string, 0, len(names))
	for _, n := range names {
		escaped = append(escaped, regexpEscape(n))
	}
	return "(" + strings.Join(escaped, "|") + ")"
}

func regexpEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(`\.+*?()|[]{}^$`, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// TriggerOnDeploy runs every suite that asked to run when something ships.
//
// A test suite that nobody remembers to click is not part of the pipeline. The
// deployed URL becomes the target, so the suite checks what was just released
// rather than whatever it was last pointed at.
func (s *Service) TriggerOnDeploy(ctx context.Context, projectID, target, actor string) (int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, COALESCE(runner_id,'') FROM test_suites
		  WHERE project_id = $1 AND run_on_deploy`, projectID)
	if err != nil {
		return 0, fmt.Errorf("testlab: suites for deploy: %w", err)
	}
	type pending struct{ suiteID, runnerID string }
	var todo []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.suiteID, &p.runnerID); err != nil {
			rows.Close()
			return 0, err
		}
		todo = append(todo, p)
	}
	rows.Close()

	started := 0
	for _, p := range todo {
		runnerID := p.runnerID
		if runnerID == "" {
			// A suite with no runner of its own uses the project's recorded
			// one, which is the common case for a smoke check.
			runner, err := s.RecordedRunner(ctx, projectID)
			if err != nil {
				continue
			}
			runnerID = runner.ID
		}
		if _, err := s.Trigger(ctx, runnerID, projectID, "deploy", actor,
			TriggerOptions{SuiteID: p.suiteID, Target: target}); err != nil {
			continue
		}
		started++
	}
	return started, nil
}

// ScheduledSuites returns suites with a cron expression, for the scheduler.
func (s *Service) ScheduledSuites(ctx context.Context, projectID string) ([]*Selection, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, tags, COALESCE(runner_id,''), default_target,
		        run_on_deploy, COALESCE(cron,'')
		   FROM test_suites WHERE cron IS NOT NULL AND cron != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*Selection
	for rows.Next() {
		sel, err := scanSelection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sel)
	}
	return out, nil
}

// HistoryEntry is one appearance of a test in a run.
type HistoryEntry struct {
	RunID          string    `json:"runId"`
	Status         string    `json:"status"`
	DurationMs     int64     `json:"durationMs"`
	Flaky          bool      `json:"flaky"`
	FailureMessage string    `json:"failureMessage,omitempty"`
	FailureDetails string    `json:"failureDetails,omitempty"`
	TargetURL      string    `json:"targetUrl,omitempty"`
	TriggerType    string    `json:"triggerType,omitempty"`
	At             time.Time `json:"at"`
	// VideoID is the recording of this attempt, when the runner left one.
	VideoID string `json:"videoId,omitempty"`
}

// TestHistory returns what one test did across runs, newest first, with the
// recording each run left behind.
func (s *Service) TestHistory(ctx context.Context, testID, projectID string) ([]*HistoryEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.run_id, c.status, c.duration_ms, c.flaky,
		        COALESCE(c.failure_message,''), COALESCE(c.failure_details,''),
		        COALESCE(r.target_url,''), r.trigger_type, c.created_at,
		        COALESCE((SELECT a.id FROM test_artifacts a
		                   WHERE a.case_id = c.id AND a.kind = 'video' LIMIT 1), '')
		   FROM test_cases c
		   JOIN test_runs r ON r.id = c.run_id
		  WHERE c.test_id = $1 AND c.project_id = $2
		  ORDER BY c.created_at DESC LIMIT 50`, testID, projectID)
	if err != nil {
		return nil, fmt.Errorf("testlab: test history: %w", err)
	}
	defer rows.Close()

	var out []*HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(&e.RunID, &e.Status, &e.DurationMs, &e.Flaky,
			&e.FailureMessage, &e.FailureDetails, &e.TargetURL, &e.TriggerType,
			&e.At, &e.VideoID); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, nil
}
