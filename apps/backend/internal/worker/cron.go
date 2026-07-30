package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/cronx"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/messaging"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/testlab"
	"github.com/mittolabs/applad/internal/workflows"
	"github.com/redis/go-redis/v9"
)

// Cron fires deploy target and function executions on a schedule.
type Cron struct {
	cfg   *config.Config
	queue *queue.Queue
	rdb   *redis.Client
	db    *db.DB
	msg   *messaging.Service
}

func NewCron(cfg *config.Config) *Cron {
	return &Cron{cfg: cfg}
}

func (w *Cron) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.rdb = rdb
	w.queue = queue.New(rdb)
	StartRedisHeartbeat(ctx, rdb, "cron")
	// Touch the file liveness heartbeat before the (up to a minute) wait for the
	// top of the minute, so the compose healthcheck's start_period does not see
	// a cron worker as hung before its first tick.
	Heartbeat()

	database, err := db.Connect(w.cfg.DatabaseDSN, w.cfg.DBMaxOpenConns, w.cfg.DBMaxIdleConns)
	if err != nil {
		return err
	}
	w.db = database
	w.msg = messagingService(w.cfg, database)

	slog.Info("cron worker: starting")

	// Wait until the top of the next minute, then tick every minute.
	now := time.Now()
	delay := time.Duration(60-now.Second())*time.Second - time.Duration(now.Nanosecond())
	select {
	case <-ctx.Done():
		return nil
	case <-time.After(delay):
	}

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	w.tick(ctx)
	Heartbeat()

	for {
		select {
		case <-ctx.Done():
			slog.Info("cron worker: shutting down")
			return nil
		case <-ticker.C:
			w.tick(ctx)
			// Written every tick (even when another replica holds the lock and
			// tick returns early) so the file heartbeat reflects this process
			// being alive, not whether it won the lock this minute.
			Heartbeat()
		}
	}
}

// schedulable is one thing that runs on a schedule, whatever kind it is.
type schedulable struct {
	kind      string // "workflow" | "function" | "deploy_target"
	id        string
	projectID string
	expr      string
}

// collect gathers everything currently scheduled across the three kinds.
func (w *Cron) collect(ctx context.Context) []schedulable {
	var out []schedulable

	queries := []struct {
		kind string
		sql  string
	}{
		{"deploy_target", `SELECT id, project_id, cron FROM deploy_targets WHERE cron IS NOT NULL AND cron != ''`},
		{"function", `SELECT id, project_id, cron FROM functions WHERE cron IS NOT NULL AND cron != '' AND status = 'active'`},
	}
	for _, q := range queries {
		rows, err := w.db.QueryContext(ctx, q.sql)
		if err != nil {
			slog.Error("cron worker: query failed", "kind", q.kind, "error", err)
			continue
		}
		for rows.Next() {
			var s schedulable
			s.kind = q.kind
			if err := rows.Scan(&s.id, &s.projectID, &s.expr); err == nil {
				out = append(out, s)
			}
		}
		rows.Close()
	}

	// Test suites schedule themselves the same way anything else does.
	if rows, err := w.db.QueryContext(ctx,
		`SELECT id, project_id, cron FROM test_suites WHERE cron IS NOT NULL AND cron != ''`); err == nil {
		for rows.Next() {
			var s schedulable
			s.kind = "test_suite"
			if err := rows.Scan(&s.id, &s.projectID, &s.expr); err == nil {
				out = append(out, s)
			}
		}
		rows.Close()
	}

	// Workflows keep the expression inside their trigger config.
	rows, err := w.db.QueryContext(ctx,
		`SELECT id, project_id, trigger_config FROM workflows WHERE trigger_type = 'cron' AND status = 'active'`)
	if err != nil {
		slog.Error("cron worker: query failed", "kind", "workflow", "error", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, projectID string
		var tcJSON []byte
		if err := rows.Scan(&id, &projectID, &tcJSON); err != nil {
			continue
		}
		var tc map[string]interface{}
		json.Unmarshal(tcJSON, &tc) //nolint:errcheck
		expr, _ := tc["cron"].(string)
		if expr != "" {
			out = append(out, schedulable{kind: "workflow", id: id, projectID: projectID, expr: expr})
		}
	}
	return out
}

func (w *Cron) tick(ctx context.Context) {
	// One instance fires at a time. Without this every replica would run every
	// job. The lock is held slightly less than the tick interval so a crashed
	// worker cannot block the next one for long.
	lock, err := w.rdb.SetNX(ctx, "applad:cron:lock", time.Now().UTC().String(), 55*time.Second).Result()
	if err != nil {
		slog.Error("cron worker: lock error", "error", err)
		return
	}
	if !lock {
		return // another instance is ticking
	}

	now := time.Now().UTC()
	for _, s := range w.collect(ctx) {
		w.evaluate(ctx, s, now)
	}

	w.sweepStaleReleases(ctx)
	w.sweepScheduledMessages(ctx)
}

// sweepScheduledMessages delivers recorded messages whose scheduled_at has
// arrived. Immediate sends happen inline on create; a message given a future
// scheduledAt is stored 'scheduled' and waits here for its minute.
func (w *Cron) sweepScheduledMessages(ctx context.Context) {
	if w.msg == nil {
		return
	}
	n, err := w.msg.SweepScheduledMessages(ctx, nil)
	if err != nil {
		slog.Error("cron worker: scheduled message sweep failed", "error", err)
		return
	}
	if n > 0 {
		slog.Info("cron worker: delivered scheduled messages", "count", n)
	}
}

// messagingService builds a messaging service from instance config, mirroring
// the wiring in router.go so the sweep sends through the same providers.
func messagingService(cfg *config.Config, database *db.DB) *messaging.Service {
	return messaging.NewService(database, messaging.Config{
		Host:            cfg.SMTPHost,
		Port:            cfg.SMTPPort,
		Username:        cfg.SMTPUser,
		Password:        cfg.SMTPPass,
		From:            cfg.SMTPFrom,
		TwilioSID:       cfg.TwilioSID,
		TwilioToken:     cfg.TwilioToken,
		TwilioFrom:      cfg.TwilioFrom,
		FCMServerKey:    cfg.FCMServerKey,
		MailgunAPIKey:   cfg.MailgunAPIKey,
		MailgunDomain:   cfg.MailgunDomain,
		ResendAPIKey:    cfg.ResendAPIKey,
		VonageAPIKey:    cfg.VonageAPIKey,
		VonageAPISecret: cfg.VonageAPISecret,
		VonageFrom:      cfg.VonageFrom,
		MSG91AuthKey:    cfg.MSG91AuthKey,
		MSG91SenderID:   cfg.MSG91SenderID,
		APNSKeyID:       cfg.APNSKeyID,
		APNSTeamID:      cfg.APNSTeamID,
		APNSKeyPath:     cfg.APNSKeyPath,
		APNSBundleID:    cfg.APNSBundleID,
	})
}

// sweepStaleReleases fails releases whose worker died mid-build. Without it a
// lost builds worker leaves rows 'building' forever and the console shows a
// deploy that never ends. deploy_releases has no updated_at, so age is judged
// from when the build started (or the row was created); an hour is far past
// any build timeout.
func (w *Cron) sweepStaleReleases(ctx context.Context) {
	res, err := w.db.ExecContext(ctx,
		`UPDATE deploy_releases
		    SET status = 'failed', error = 'worker lost', completed_at = NOW()
		  WHERE status IN ('building', 'deploying')
		    AND COALESCE(started_at, created_at) < NOW() - INTERVAL '1 hour'`)
	if err != nil {
		slog.Error("cron worker: stale release sweep failed", "error", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		slog.Warn("cron worker: failed stale releases", "count", n)
	}
}

// cronOutcome is what evaluate should do with one schedulable after comparing
// its stored state to the clock. It is computed by decideCron, which is pure so
// the scheduling rules are testable without a database.
type cronOutcome int

const (
	cronInit   cronOutcome = iota // new or changed schedule: set next_run, do not fire
	cronNotDue                    // next_run is still in the future: nothing to do
	cronClaim                     // due: try to claim the occurrence, and fire if won
)

// cronDecision carries the outcome plus the values evaluate acts on.
type cronDecision struct {
	outcome cronOutcome
	nextRun time.Time // the next scheduled run; set for cronInit and cronClaim
	missed  int       // occurrences skipped while the worker was down (cronClaim only)
}

// decideCron is the pure heart of the scheduler: given a parsed schedule, the
// stored state for an entity and the current time, it decides whether the
// entity is due, how many runs were missed, and when it should next run —
// without touching the database.
//
// A missed backlog is collapsed to a single run: several occurrences that came
// and went while the worker was down fire once and resync to the next future
// time rather than replaying every occurrence (a nightly job after a week of
// downtime yields one run, not seven).
func decideCron(schedule cronx.Schedule, rowExists bool, storedExpr, currentExpr string, storedNext sql.NullTime, now time.Time) cronDecision {
	// New schedule, or the expression changed: start from now rather than firing
	// immediately for a window nobody asked for. A row without a next_run_at yet
	// is seeded the same way, still without firing.
	if !rowExists || storedExpr != currentExpr || !storedNext.Valid {
		return cronDecision{outcome: cronInit, nextRun: schedule.Next(now)}
	}
	if storedNext.Time.After(now) {
		return cronDecision{outcome: cronNotDue}
	}

	// Due. Count the occurrences that elapsed while the worker was down so the
	// backlog is recorded, but fire only once and resync forward.
	missed := 0
	for t := schedule.Next(storedNext.Time); !t.After(now); t = schedule.Next(t) {
		missed++
		if missed > 1000 {
			break
		}
	}
	return cronDecision{outcome: cronClaim, nextRun: schedule.Next(now), missed: missed}
}

// evaluate decides whether one scheduled thing is due, fires it if so, and
// records when it should next run.
func (w *Cron) evaluate(ctx context.Context, s schedulable, now time.Time) {
	schedule, parseErr := cronx.Parse(s.expr)
	if parseErr != nil {
		// Record it rather than failing silently: an unparseable expression
		// used to mean the job simply never ran, with nothing to look at.
		w.saveState(ctx, s, nil, nil, parseErr.Error(), 0)
		return
	}

	var storedExpr string
	var storedNext, storedLast sql.NullTime
	err := w.db.QueryRowContext(ctx,
		`SELECT expression, last_run_at, next_run_at FROM cron_state WHERE kind = $1 AND entity_id = $2`,
		s.kind, s.id).Scan(&storedExpr, &storedLast, &storedNext)
	if err != nil && err != sql.ErrNoRows {
		slog.Error("cron worker: state read failed", "kind", s.kind, "id", s.id, "error", err)
		return
	}
	rowExists := err != sql.ErrNoRows

	d := decideCron(schedule, rowExists, storedExpr, s.expr, storedNext, now)
	switch d.outcome {
	case cronInit:
		// saveState's COALESCE preserves any existing last_run_at, so a nil here
		// never clobbers it.
		next := d.nextRun
		w.saveState(ctx, s, nil, &next, "", 0)
	case cronNotDue:
		return
	case cronClaim:
		if d.missed > 0 {
			slog.Warn("cron worker: missed runs while down, firing once",
				"kind", s.kind, "id", s.id, "missed", d.missed)
		}
		if w.claim(ctx, s, storedNext.Time, d.nextRun, d.missed, now) {
			w.fire(ctx, s)
		}
	}
}

// claim records a run against cron_state, conditional on next_run_at still being
// what we read. The database arbitrates, so two workers cannot both fire the
// same occurrence even if one overruns its lock: the loser updates zero rows and
// this returns false. Returning true means this worker owns the occurrence and
// should fire it.
func (w *Cron) claim(ctx context.Context, s schedulable, prevNext, nextRun time.Time, missed int, now time.Time) bool {
	res, err := w.db.ExecContext(ctx,
		`UPDATE cron_state
		    SET last_run_at = $1, next_run_at = $2, missed_runs = missed_runs + $3, updated_at = NOW()
		  WHERE kind = $4 AND entity_id = $5 AND next_run_at = $6`,
		now, nextRun, missed, s.kind, s.id, prevNext)
	if err != nil {
		slog.Error("cron worker: claim failed", "kind", s.kind, "id", s.id, "error", err)
		return false
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false // another instance claimed this occurrence
	}
	return true
}

func (w *Cron) saveState(ctx context.Context, s schedulable, lastRun, nextRun *time.Time, parseErr string, missed int) {
	_, err := w.db.ExecContext(ctx,
		`INSERT INTO cron_state (kind, entity_id, project_id, expression, parse_error, last_run_at, next_run_at, missed_runs, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, NOW(), NOW())
		 ON CONFLICT (kind, entity_id) DO UPDATE SET
		   project_id = EXCLUDED.project_id,
		   expression = EXCLUDED.expression,
		   parse_error = EXCLUDED.parse_error,
		   last_run_at = COALESCE(EXCLUDED.last_run_at, cron_state.last_run_at),
		   next_run_at = EXCLUDED.next_run_at,
		   missed_runs = cron_state.missed_runs + EXCLUDED.missed_runs,
		   updated_at = NOW()`,
		s.kind, s.id, s.projectID, s.expr, parseErr, lastRun, nextRun, missed)
	if err != nil {
		slog.Error("cron worker: state write failed", "kind", s.kind, "id", s.id, "error", err)
	}
}

func (w *Cron) fire(ctx context.Context, s schedulable) {
	slog.Info("cron worker: firing", "kind", s.kind, "id", s.id, "expr", s.expr)
	switch s.kind {
	case "deploy_target":
		w.fireTarget(ctx, s.id, s.projectID)
	case "function":
		w.fireFunction(ctx, s.id, s.projectID)
	case "test_suite":
		svc := testlab.NewService(w.db, w.queue)
		sel, err := svc.GetSelection(ctx, s.id, s.projectID)
		if err != nil {
			slog.Error("cron worker: suite lookup failed", "suite_id", s.id, "error", err)
			return
		}
		runnerID := sel.RunnerID
		if runnerID == "" {
			if runner, err := svc.RecordedRunner(ctx, s.projectID); err == nil {
				runnerID = runner.ID
			}
		}
		if _, err := svc.Trigger(ctx, runnerID, s.projectID, "schedule", "cron",
			testlab.TriggerOptions{SuiteID: sel.ID}); err != nil {
			slog.Error("cron worker: fire test suite failed", "suite_id", s.id, "error", err)
		}

	case "workflow":
		svc := workflows.NewService(w.db, w.queue)
		if _, err := svc.Execute(ctx, s.id, s.projectID, map[string]interface{}{"trigger": "cron"}); err != nil {
			slog.Error("cron worker: fire workflow failed", "workflow_id", s.id, "error", err)
		}
	}
}

func (w *Cron) fireTarget(ctx context.Context, targetID, projectID string) {
	jobID := targetID + "-cron-" + time.Now().UTC().Format("20060102150405")
	if err := w.queue.Push(ctx, "builds", queue.Job{
		ID: jobID, Type: "deploy_execution",
		Payload: map[string]interface{}{"targetId": targetID, "projectId": projectID, "trigger": "cron"},
	}); err != nil {
		slog.Error("cron worker: push deploy job failed", "target_id", targetID, "error", err)
	}
}

func (w *Cron) fireFunction(ctx context.Context, functionID, projectID string) {
	jobID := functionID + "-cron-" + time.Now().UTC().Format("20060102150405")
	if err := w.queue.Push(ctx, "executions", queue.Job{
		ID: jobID, Type: "function_execution",
		Payload: map[string]interface{}{"functionId": functionID, "projectId": projectID, "trigger": "cron"},
	}); err != nil {
		slog.Error("cron worker: push function job failed", "function_id", functionID, "error", err)
	}
}
