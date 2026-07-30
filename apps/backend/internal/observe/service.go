// Package observe implements the Observe API: error tracking, logs,
// performance metrics, releases, session replays, uptime monitors,
// cron monitors, and alert rules.
package observe

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// ── Models ───────────────────────────────────────────────────────────────────

type Error struct {
	ID            string           `json:"$id"`
	ProjectID     string           `json:"projectId"`
	Title         string           `json:"title"`
	ErrorType     string           `json:"errorType"`
	Level         string           `json:"level"`
	Status        string           `json:"status"`
	Fingerprint   string           `json:"fingerprint"`
	StackTrace    string           `json:"stackTrace"`
	Breadcrumbs   []map[string]any `json:"breadcrumbs"`
	UserContext   map[string]any   `json:"userContext"`
	RequestCtx    map[string]any   `json:"requestContext"`
	RuntimeCtx    map[string]any   `json:"runtimeContext"`
	Tags          map[string]any   `json:"tags"`
	Environment   string           `json:"environment"`
	Release       string           `json:"release"`
	Count         int64            `json:"count"`
	AffectedUsers int64            `json:"affectedUsers"`
	Priority      string           `json:"priority"`
	Assignee      string           `json:"assignee"`
	Activity      []map[string]any `json:"activity,omitempty"`
	FirstSeen     time.Time        `json:"firstSeen"`
	LastSeen      time.Time        `json:"lastSeen"`
}

type LogEntry struct {
	ID          string         `json:"$id"`
	ProjectID   string         `json:"projectId"`
	Level       string         `json:"level"`
	Message     string         `json:"message"`
	Source      string         `json:"source"`
	Environment string         `json:"environment"`
	Release     string         `json:"release"`
	Meta        map[string]any `json:"meta"`
	TraceID     string         `json:"traceId"`
	SpanID      string         `json:"spanId"`
	CreatedAt   time.Time      `json:"$createdAt"`
}

type PerfSnapshot struct {
	ID         string    `json:"$id"`
	ProjectID  string    `json:"projectId"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	P50Ms      float64   `json:"p50Ms"`
	P75Ms      float64   `json:"p75Ms"`
	P95Ms      float64   `json:"p95Ms"`
	P99Ms      float64   `json:"p99Ms"`
	RPS        float64   `json:"rps"`
	ErrorPct   float64   `json:"errorPct"`
	ReqCount   int64     `json:"reqCount"`
	BucketHour time.Time `json:"bucketHour"`
}

type WebVitals struct {
	ID        string    `json:"$id"`
	ProjectID string    `json:"projectId"`
	PageURL   string    `json:"pageUrl"`
	LCP       *float64  `json:"lcp"`
	FID       *float64  `json:"fid"`
	CLS       *float64  `json:"cls"`
	TTFB      *float64  `json:"ttfb"`
	FCP       *float64  `json:"fcp"`
	INP       *float64  `json:"inp"`
	CreatedAt time.Time `json:"$createdAt"`
}

type Release struct {
	ID                   string           `json:"$id"`
	ProjectID            string           `json:"projectId"`
	Version              string           `json:"version"`
	Environment          string           `json:"environment"`
	CommitCount          int              `json:"commitCount"`
	Commits              []map[string]any `json:"commits"`
	CrashFreeSessionsPct float64          `json:"crashFreeSessionsPct"`
	NewIssues            int              `json:"newIssues"`
	RegressedIssues      int              `json:"regressedIssues"`
	FixedIssues          int              `json:"fixedIssues"`
	CreatedAt            time.Time        `json:"$createdAt"`
	DeployedAt           *time.Time       `json:"deployedAt"`
}

type Replay struct {
	ID           string           `json:"$id"`
	ProjectID    string           `json:"projectId"`
	SessionID    string           `json:"sessionId"`
	UserID       string           `json:"userId"`
	User         string           `json:"user"`
	URL          string           `json:"url"`
	Browser      string           `json:"browser"`
	OS           string           `json:"os"`
	Country      string           `json:"country"`
	DurationSecs int              `json:"durationSecs"`
	ErrorCount   int              `json:"errorCount"`
	HasRageClick bool             `json:"hasRageClick"`
	HasDeadClick bool             `json:"hasDeadClick"`
	Events       []map[string]any `json:"events,omitempty"`
	Network      []map[string]any `json:"network,omitempty"`
	Console      []map[string]any `json:"console,omitempty"`
	StartedAt    time.Time        `json:"startedAt"`
	EndedAt      *time.Time       `json:"endedAt"`
}

type UptimeMonitor struct {
	ID           string `json:"$id"`
	ProjectID    string `json:"projectId"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	CheckType    string `json:"checkType"`
	IntervalSecs int    `json:"intervalSecs"`
	Keyword      string `json:"keyword"`
	Status       string `json:"status"`
	Enabled      bool   `json:"enabled"`
	// Nil until the first check: a monitor that has never run has no uptime,
	// and 100.0 is not a safe stand-in for "unknown".
	UptimePct *float64 `json:"uptimePct"`
	// Nil for the same reason as UptimePct: 0ms is a measurement, not a gap.
	LatencyMs   *int       `json:"latencyMs"`
	LastChecked *time.Time `json:"lastChecked"`
	History     []string   `json:"history"`
	CreatedAt   time.Time  `json:"$createdAt"`
}

type CronMonitor struct {
	ID             string     `json:"$id"`
	ProjectID      string     `json:"projectId"`
	Name           string     `json:"name"`
	Schedule       string     `json:"schedule"`
	Timezone       string     `json:"timezone"`
	GracePeriod    int        `json:"gracePeriod"`
	Status         string     `json:"status"`
	Enabled        bool       `json:"enabled"`
	LastDurationMs *int       `json:"lastDurationMs"`
	LastRunAt      *time.Time `json:"lastRunAt"`
	NextRunAt      *time.Time `json:"nextRunAt"`
	History        []string   `json:"history"`
	CreatedAt      time.Time  `json:"$createdAt"`
}

type AlertRule struct {
	ID        string     `json:"$id"`
	ProjectID string     `json:"projectId"`
	Name      string     `json:"name"`
	Metric    string     `json:"metric"`
	Operator  string     `json:"operator"`
	Threshold float64    `json:"threshold"`
	Window    string     `json:"window"`
	Severity  string     `json:"severity"`
	Channel   string     `json:"channel"`
	Enabled   bool       `json:"enabled"`
	LastFired *time.Time `json:"lastFired"`
	CreatedAt time.Time  `json:"$createdAt"`
}

type AlertIncident struct {
	ID         string     `json:"$id"`
	RuleID     string     `json:"ruleId"`
	ProjectID  string     `json:"projectId"`
	RuleName   string     `json:"ruleName"`
	Severity   string     `json:"severity"`
	Value      float64    `json:"value"`
	FiredAt    time.Time  `json:"firedAt"`
	ResolvedAt *time.Time `json:"resolvedAt"`
}

// ── Service ──────────────────────────────────────────────────────────────────

type Service struct {
	db *db.DB
}

func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// jsonb helpers
func toJSONB(v any) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(v)
}
func fromJSONB(data []byte, dst any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, dst)
}
func sliceJSONB(data []byte) []map[string]any {
	var out []map[string]any
	_ = json.Unmarshal(data, &out)
	return out
}
func mapJSONB(data []byte) map[string]any {
	out := map[string]any{}
	_ = json.Unmarshal(data, &out)
	return out
}

// ── Errors ───────────────────────────────────────────────────────────────────

func fingerprint(projectID, title, errType string) string {
	h := md5.Sum([]byte(projectID + "|" + title + "|" + errType))
	return fmt.Sprintf("%x", h)
}

// CaptureError ingests an error event. If the same fingerprint already exists,
// it increments the counter and updates last_seen; otherwise it inserts.
func (s *Service) CaptureError(ctx context.Context, projectID string, req CaptureErrorRequest) (*Error, error) {
	fp := fingerprint(projectID, req.Title, req.ErrorType)
	id := uid.New("unique()")
	now := time.Now().UTC()

	breadcrumbsJSON, _ := toJSONB(req.Breadcrumbs)
	userCtxJSON, _ := toJSONB(req.UserContext)
	reqCtxJSON, _ := toJSONB(req.RequestContext)
	runtimeJSON, _ := toJSONB(req.RuntimeContext)
	tagsJSON, _ := toJSONB(req.Tags)

	stack := req.StackTrace
	level := req.Level
	if level == "" {
		level = "error"
	}
	env := req.Environment
	if env == "" {
		env = "production"
	}

	const q = `
INSERT INTO observe_errors
    (id, project_id, title, error_type, level, status, fingerprint, stack_trace,
     breadcrumbs, user_context, request_ctx, runtime_ctx, tags,
     environment, release, count, affected_users, first_seen, last_seen)
VALUES ($1,$2,$3,$4,$5,'unresolved',$6,$7,$8,$9,$10,$11,$12,$13,$14,1,0,$15,$15)
ON CONFLICT (project_id, fingerprint) DO UPDATE SET
    count        = observe_errors.count + 1,
    last_seen    = EXCLUDED.last_seen,
    stack_trace  = CASE WHEN EXCLUDED.stack_trace <> '' THEN EXCLUDED.stack_trace ELSE observe_errors.stack_trace END,
    breadcrumbs  = CASE WHEN EXCLUDED.breadcrumbs::text <> '[]' THEN EXCLUDED.breadcrumbs ELSE observe_errors.breadcrumbs END
RETURNING id, count, first_seen, last_seen`

	var retID string
	var count int64
	var firstSeen, lastSeen time.Time
	err := s.db.QueryRowContext(ctx, q,
		id, projectID, req.Title, req.ErrorType, level, fp, stack,
		breadcrumbsJSON, userCtxJSON, reqCtxJSON, runtimeJSON, tagsJSON,
		env, req.Release, now,
	).Scan(&retID, &count, &firstSeen, &lastSeen)
	if err != nil {
		return nil, fmt.Errorf("observe: capture error: %w", err)
	}

	return &Error{
		ID: retID, ProjectID: projectID, Title: req.Title,
		ErrorType: req.ErrorType, Level: level, Status: "unresolved",
		Fingerprint: fp, StackTrace: stack,
		Breadcrumbs: req.Breadcrumbs, UserContext: req.UserContext,
		RequestCtx: req.RequestContext, RuntimeCtx: req.RuntimeContext,
		Tags: req.Tags, Environment: env, Release: req.Release,
		Count: count, FirstSeen: firstSeen, LastSeen: lastSeen,
	}, nil
}

func (s *Service) ListErrors(ctx context.Context, projectID, status, level, search string, limit int) ([]Error, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	args := []any{projectID}
	where := "project_id = $1"
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if level != "" {
		args = append(args, level)
		where += fmt.Sprintf(" AND level = $%d", len(args))
	}
	if search != "" {
		args = append(args, "%"+search+"%")
		where += fmt.Sprintf(" AND (title ILIKE $%d OR error_type ILIKE $%d OR stack_trace ILIKE $%d)", len(args), len(args), len(args))
	}
	args = append(args, limit)

	q := fmt.Sprintf(`
SELECT id, project_id, title, error_type, level, status, fingerprint,
       stack_trace, breadcrumbs, user_context, request_ctx, runtime_ctx, tags,
       environment, release, count, affected_users, priority, assignee, first_seen, last_seen
FROM observe_errors
WHERE %s
ORDER BY last_seen DESC
LIMIT $%d`, where, len(args))

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Error
	for rows.Next() {
		var e Error
		var bc, uc, rc, rtc, tags []byte
		if err := rows.Scan(
			&e.ID, &e.ProjectID, &e.Title, &e.ErrorType, &e.Level, &e.Status,
			&e.Fingerprint, &e.StackTrace, &bc, &uc, &rc, &rtc, &tags,
			&e.Environment, &e.Release, &e.Count, &e.AffectedUsers,
			&e.Priority, &e.Assignee, &e.FirstSeen, &e.LastSeen,
		); err != nil {
			return nil, err
		}
		e.Breadcrumbs = sliceJSONB(bc)
		e.UserContext = mapJSONB(uc)
		e.RequestCtx = mapJSONB(rc)
		e.RuntimeCtx = mapJSONB(rtc)
		e.Tags = mapJSONB(tags)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Service) GetError(ctx context.Context, projectID, errorID string) (*Error, error) {
	var e Error
	var bc, uc, rc, rtc, tags []byte
	err := s.db.QueryRowContext(ctx, `
SELECT id, project_id, title, error_type, level, status, fingerprint,
       stack_trace, breadcrumbs, user_context, request_ctx, runtime_ctx, tags,
       environment, release, count, affected_users, priority, assignee, first_seen, last_seen
FROM observe_errors WHERE id = $1 AND project_id = $2`, errorID, projectID).Scan(
		&e.ID, &e.ProjectID, &e.Title, &e.ErrorType, &e.Level, &e.Status,
		&e.Fingerprint, &e.StackTrace, &bc, &uc, &rc, &rtc, &tags,
		&e.Environment, &e.Release, &e.Count, &e.AffectedUsers,
		&e.Priority, &e.Assignee, &e.FirstSeen, &e.LastSeen,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	e.Breadcrumbs = sliceJSONB(bc)
	e.UserContext = mapJSONB(uc)
	e.RequestCtx = mapJSONB(rc)
	e.RuntimeCtx = mapJSONB(rtc)
	e.Tags = mapJSONB(tags)

	// load activity
	activity, _ := s.listErrorActivity(ctx, errorID)
	e.Activity = activity
	return &e, nil
}

func (s *Service) UpdateErrorStatus(ctx context.Context, projectID, errorID, status, actorName string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE observe_errors SET status = $1 WHERE id = $2 AND project_id = $3`,
		status, errorID, projectID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	_, _ = s.db.ExecContext(ctx,
		`INSERT INTO observe_error_activity (id, error_id, type, user_name) VALUES ($1,$2,$3,$4)`,
		uid.New("unique()"), errorID, status, actorName)
	return nil
}

func (s *Service) listErrorActivity(ctx context.Context, errorID string) ([]map[string]any, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, type, user_name, text, created_at FROM observe_error_activity
         WHERE error_id = $1 ORDER BY created_at DESC LIMIT 20`, errorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, typ, user, text string
		var ts time.Time
		if err := rows.Scan(&id, &typ, &user, &text, &ts); err != nil {
			continue
		}
		out = append(out, map[string]any{
			"$id": id, "type": typ, "user": user,
			"text": text, "timestamp": ts,
		})
	}
	return out, rows.Err()
}

// ── Logs ─────────────────────────────────────────────────────────────────────

func (s *Service) CaptureLog(ctx context.Context, projectID string, req CaptureLogRequest) (*LogEntry, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()
	metaJSON, _ := toJSONB(req.Meta)
	level := req.Level
	if level == "" {
		level = "info"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO observe_logs (id, project_id, level, message, source, environment, release, meta, trace_id, span_id, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		id, projectID, level, req.Message, req.Source,
		req.Environment, req.Release, metaJSON, req.TraceID, req.SpanID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("observe: capture log: %w", err)
	}
	return &LogEntry{
		ID: id, ProjectID: projectID, Level: level,
		Message: req.Message, Source: req.Source,
		Environment: req.Environment, Release: req.Release,
		Meta: req.Meta, TraceID: req.TraceID, SpanID: req.SpanID,
		CreatedAt: now,
	}, nil
}

func (s *Service) ListLogs(ctx context.Context, projectID, level, source string, limit int) ([]LogEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	args := []any{projectID}
	where := "project_id = $1"
	if level != "" {
		args = append(args, level)
		where += fmt.Sprintf(" AND level = $%d", len(args))
	}
	if source != "" {
		args = append(args, source)
		where += fmt.Sprintf(" AND source = $%d", len(args))
	}
	args = append(args, limit)

	q := fmt.Sprintf(`
SELECT id, project_id, level, message, source, environment, release, meta, trace_id, span_id, created_at
FROM observe_logs WHERE %s ORDER BY created_at DESC LIMIT $%d`, where, len(args))

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LogEntry
	for rows.Next() {
		var e LogEntry
		var metaB []byte
		if err := rows.Scan(
			&e.ID, &e.ProjectID, &e.Level, &e.Message, &e.Source,
			&e.Environment, &e.Release, &metaB, &e.TraceID, &e.SpanID, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		e.Meta = mapJSONB(metaB)
		out = append(out, e)
	}
	return out, rows.Err()
}

// ── Performance ───────────────────────────────────────────────────────────────

func (s *Service) RecordPerf(ctx context.Context, projectID string, req RecordPerfRequest) error {
	id := uid.New("unique()")
	bucket := time.Now().UTC().Truncate(time.Hour)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO observe_perf_snapshots
    (id, project_id, method, path, p50_ms, p75_ms, p95_ms, p99_ms, rps, error_pct, req_count, bucket_hour)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT DO NOTHING`,
		id, projectID, req.Method, req.Path,
		req.P50Ms, req.P75Ms, req.P95Ms, req.P99Ms,
		req.RPS, req.ErrorPct, req.ReqCount, bucket,
	)
	return err
}

func (s *Service) GetPerformance(ctx context.Context, projectID string) (map[string]any, error) {
	// Aggregate last 24h per endpoint
	rows, err := s.db.QueryContext(ctx, `
SELECT method, path,
       AVG(p50_ms), AVG(p75_ms), AVG(p95_ms), AVG(p99_ms),
       AVG(rps), AVG(error_pct), SUM(req_count)
FROM observe_perf_snapshots
WHERE project_id = $1 AND bucket_hour >= NOW() - INTERVAL '24 hours'
GROUP BY method, path
ORDER BY AVG(p95_ms) DESC
LIMIT 50`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []map[string]any
	var totalP50, totalP95, totalP99 float64
	var endpointCount int
	for rows.Next() {
		var method, path string
		var p50, p75, p95, p99, rps, errPct float64
		var reqCount int64
		if err := rows.Scan(&method, &path, &p50, &p75, &p95, &p99, &rps, &errPct, &reqCount); err != nil {
			continue
		}
		endpoints = append(endpoints, map[string]any{
			"method": method, "path": path,
			"p50Ms": p50, "p75Ms": p75, "p95Ms": p95, "p99Ms": p99,
			"rps": rps, "errorPct": errPct, "reqCount": reqCount,
		})
		totalP50 += p50
		totalP95 += p95
		totalP99 += p99
		endpointCount++
	}

	avgP50, avgP95, avgP99 := 0.0, 0.0, 0.0
	if endpointCount > 0 {
		avgP50 = totalP50 / float64(endpointCount)
		avgP95 = totalP95 / float64(endpointCount)
		avgP99 = totalP99 / float64(endpointCount)
	}
	// Apdex: satisfied < T (e.g. 500ms), tolerating < 4T
	T := 500.0
	apdex := 1.0
	if avgP95 > 0 {
		satisfied := 0.0
		tolerating := 0.0
		if avgP50 <= T {
			satisfied = 1.0
		} else if avgP50 <= 4*T {
			tolerating = 1.0
		}
		apdex = (satisfied + tolerating/2) / 1.0
		if apdex > 1 {
			apdex = 1
		}
	}

	// Hourly chart for last 24h
	chartRows, _ := s.db.QueryContext(ctx, `
SELECT bucket_hour, AVG(p50_ms), AVG(p95_ms), AVG(p99_ms)
FROM observe_perf_snapshots
WHERE project_id = $1 AND bucket_hour >= NOW() - INTERVAL '24 hours'
GROUP BY bucket_hour ORDER BY bucket_hour`, projectID)
	var chart []map[string]any
	if chartRows != nil {
		defer chartRows.Close()
		for chartRows.Next() {
			var bh time.Time
			var p50, p95, p99 float64
			if err := chartRows.Scan(&bh, &p50, &p95, &p99); err != nil {
				continue
			}
			chart = append(chart, map[string]any{
				"hour": bh, "p50": p50, "p95": p95, "p99": p99,
			})
		}
	}

	// Web vitals avg
	var lcp, fid, cls, ttfb, fcp, inp sql.NullFloat64
	_ = s.db.QueryRowContext(ctx, `
SELECT AVG(lcp_ms), AVG(fid_ms), AVG(cls_score), AVG(ttfb_ms), AVG(fcp_ms), AVG(inp_ms)
FROM observe_web_vitals
WHERE project_id = $1 AND created_at >= NOW() - INTERVAL '24 hours'`, projectID).
		Scan(&lcp, &fid, &cls, &ttfb, &fcp, &inp)

	vitals := map[string]any{}
	if lcp.Valid {
		vitals["lcp"] = lcp.Float64
	}
	if fid.Valid {
		vitals["fid"] = fid.Float64
	}
	if cls.Valid {
		vitals["cls"] = cls.Float64
	}
	if ttfb.Valid {
		vitals["ttfb"] = ttfb.Float64
	}
	if fcp.Valid {
		vitals["fcp"] = fcp.Float64
	}
	if inp.Valid {
		vitals["inp"] = inp.Float64
	}

	return map[string]any{
		"metrics": map[string]any{
			"p50Ms": avgP50, "p75Ms": avgP50, "p95Ms": avgP95, "p99Ms": avgP99,
			"apdex": apdex, "rps": 0,
		},
		"vitals":    vitals,
		"endpoints": endpoints,
		"chart":     chart,
	}, nil
}

func (s *Service) RecordWebVitals(ctx context.Context, projectID string, req WebVitalsRequest) error {
	id := uid.New("unique()")
	_, err := s.db.ExecContext(ctx, `
INSERT INTO observe_web_vitals (id, project_id, page_url, lcp_ms, fid_ms, cls_score, ttfb_ms, fcp_ms, inp_ms)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, projectID, req.PageURL,
		req.LCP, req.FID, req.CLS, req.TTFB, req.FCP, req.INP,
	)
	return err
}

// ── Overview ─────────────────────────────────────────────────────────────────

func (s *Service) GetOverview(ctx context.Context, projectID string) (map[string]any, error) {
	var errorsToday int64
	_ = s.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM observe_errors
WHERE project_id = $1 AND last_seen >= NOW() - INTERVAL '24 hours'`, projectID).Scan(&errorsToday)

	var logsLastHour int64
	_ = s.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM observe_logs
WHERE project_id = $1 AND created_at >= NOW() - INTERVAL '1 hour'`, projectID).Scan(&logsLastHour)

	var avgP95 sql.NullFloat64
	_ = s.db.QueryRowContext(ctx, `
SELECT AVG(p95_ms) FROM observe_perf_snapshots
WHERE project_id = $1 AND bucket_hour >= NOW() - INTERVAL '24 hours'`, projectID).Scan(&avgP95)

	var lcp, fid, cls, ttfb, fcp, inp sql.NullFloat64
	_ = s.db.QueryRowContext(ctx, `
SELECT AVG(lcp_ms), AVG(fid_ms), AVG(cls_score), AVG(ttfb_ms), AVG(fcp_ms), AVG(inp_ms)
FROM observe_web_vitals
WHERE project_id = $1 AND created_at >= NOW() - INTERVAL '24 hours'`, projectID).
		Scan(&lcp, &fid, &cls, &ttfb, &fcp, &inp)

	// Uptime — avg across monitors
	var avgUptime sql.NullFloat64
	_ = s.db.QueryRowContext(ctx, `
SELECT AVG(uptime_pct) FROM observe_uptime_monitors
WHERE project_id = $1 AND enabled = TRUE`, projectID).Scan(&avgUptime)

	p95 := 0.0
	if avgP95.Valid {
		p95 = avgP95.Float64
	}

	vitals := map[string]any{}
	if lcp.Valid {
		vitals["lcp"] = lcp.Float64
	}
	if fid.Valid {
		vitals["fid"] = fid.Float64
	}
	if cls.Valid {
		vitals["cls"] = cls.Float64
	}
	if ttfb.Valid {
		vitals["ttfb"] = ttfb.Float64
	}
	if fcp.Valid {
		vitals["fcp"] = fcp.Float64
	}
	if inp.Valid {
		vitals["inp"] = inp.Float64
	}

	// A statistic nobody has measured is left out rather than sent as its
	// perfect value. An instance with no monitors reported 100% uptime and a
	// 1.00 apdex, which are the two numbers people read as proof.
	stats := map[string]any{
		"errorsToday":  errorsToday,
		"logsLastHour": logsLastHour,
	}
	if avgP95.Valid {
		stats["p95Ms"] = p95
	}
	if avgUptime.Valid {
		stats["uptimePct"] = avgUptime.Float64
	}
	if p95 > 0 {
		switch {
		case p95 <= 500:
			stats["apdex"] = 1.0
		case p95 <= 2000:
			stats["apdex"] = 0.75
		default:
			stats["apdex"] = 0.5
		}
	}

	return map[string]any{
		"stats":    stats,
		"vitals":   vitals,
		"services": []any{},
	}, nil
}

// ── Releases ──────────────────────────────────────────────────────────────────

func (s *Service) CreateRelease(ctx context.Context, projectID string, req CreateReleaseRequest) (*Release, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()
	commitsJSON, _ := toJSONB(req.Commits)
	env := req.Environment
	if env == "" {
		env = "production"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO observe_releases
    (id, project_id, version, environment, commit_count, commits,
     crash_free_sessions_pct, created_at)
VALUES ($1,$2,$3,$4,$5,$6,100.0,$7)
ON CONFLICT (project_id, version, environment) DO UPDATE SET
    commit_count = EXCLUDED.commit_count,
    commits      = EXCLUDED.commits`,
		id, projectID, req.Version, env, len(req.Commits), commitsJSON, now,
	)
	if err != nil {
		return nil, fmt.Errorf("observe: create release: %w", err)
	}
	return &Release{
		ID: id, ProjectID: projectID, Version: req.Version,
		Environment: env, CommitCount: len(req.Commits),
		Commits: req.Commits, CrashFreeSessionsPct: 100.0, CreatedAt: now,
	}, nil
}

func (s *Service) ListReleases(ctx context.Context, projectID string) ([]Release, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, project_id, version, environment, commit_count, commits,
       crash_free_sessions_pct, new_issues, regressed_issues, fixed_issues,
       created_at, deployed_at
FROM observe_releases WHERE project_id = $1
ORDER BY created_at DESC LIMIT 50`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Release
	for rows.Next() {
		var r Release
		var commitsB []byte
		if err := rows.Scan(
			&r.ID, &r.ProjectID, &r.Version, &r.Environment, &r.CommitCount,
			&commitsB, &r.CrashFreeSessionsPct, &r.NewIssues, &r.RegressedIssues,
			&r.FixedIssues, &r.CreatedAt, &r.DeployedAt,
		); err != nil {
			return nil, err
		}
		r.Commits = sliceJSONB(commitsB)
		out = append(out, r)
	}
	return out, rows.Err()
}

// ── Replays ───────────────────────────────────────────────────────────────────

func (s *Service) CreateReplay(ctx context.Context, projectID string, req CreateReplayRequest) (*Replay, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()
	eventsJSON, _ := toJSONB(req.Events)
	networkJSON, _ := toJSONB(req.Network)
	consoleJSON, _ := toJSONB(req.Console)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO observe_replays
    (id, project_id, session_id, user_id, user_name, url, browser, os, country,
     duration_secs, error_count, has_rage_click, has_dead_click,
     events, network, console, started_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		id, projectID, req.SessionID, req.UserID, req.User, req.URL,
		req.Browser, req.OS, req.Country,
		req.DurationSecs, req.ErrorCount, req.HasRageClick, req.HasDeadClick,
		eventsJSON, networkJSON, consoleJSON, now,
	)
	if err != nil {
		return nil, fmt.Errorf("observe: create replay: %w", err)
	}
	return &Replay{ID: id, ProjectID: projectID, StartedAt: now}, nil
}

func (s *Service) ListReplays(ctx context.Context, projectID string, limit int) ([]Replay, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, project_id, session_id, user_id, user_name, url, browser, os, country,
       duration_secs, error_count, has_rage_click, has_dead_click, started_at, ended_at
FROM observe_replays WHERE project_id = $1
ORDER BY started_at DESC LIMIT $2`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Replay
	for rows.Next() {
		var r Replay
		if err := rows.Scan(
			&r.ID, &r.ProjectID, &r.SessionID, &r.UserID, &r.User,
			&r.URL, &r.Browser, &r.OS, &r.Country,
			&r.DurationSecs, &r.ErrorCount, &r.HasRageClick, &r.HasDeadClick,
			&r.StartedAt, &r.EndedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) GetReplay(ctx context.Context, projectID, replayID string) (*Replay, error) {
	var r Replay
	var eventsB, networkB, consoleB []byte
	err := s.db.QueryRowContext(ctx, `
SELECT id, project_id, session_id, user_id, user_name, url, browser, os, country,
       duration_secs, error_count, has_rage_click, has_dead_click,
       events, network, console, started_at, ended_at
FROM observe_replays WHERE id = $1 AND project_id = $2`, replayID, projectID).Scan(
		&r.ID, &r.ProjectID, &r.SessionID, &r.UserID, &r.User,
		&r.URL, &r.Browser, &r.OS, &r.Country,
		&r.DurationSecs, &r.ErrorCount, &r.HasRageClick, &r.HasDeadClick,
		&eventsB, &networkB, &consoleB, &r.StartedAt, &r.EndedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Events = sliceJSONB(eventsB)
	r.Network = sliceJSONB(networkB)
	r.Console = sliceJSONB(consoleB)
	return &r, nil
}

// ── Uptime ────────────────────────────────────────────────────────────────────

func (s *Service) CreateUptimeMonitor(ctx context.Context, projectID string, req CreateUptimeMonitorRequest) (*UptimeMonitor, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO observe_uptime_monitors
    (id, project_id, name, url, check_type, interval_secs, keyword, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, projectID, req.Name, req.URL, req.CheckType,
		req.IntervalSecs, req.Keyword, now,
	)
	if err != nil {
		return nil, fmt.Errorf("observe: create monitor: %w", err)
	}
	return &UptimeMonitor{
		ID: id, ProjectID: projectID, Name: req.Name, URL: req.URL,
		CheckType: req.CheckType, IntervalSecs: req.IntervalSecs,
		// Never checked yet: saying "up" here is a guess that happens to be
		// about a URL nobody has fetched.
		Status: "pending", Enabled: true, CreatedAt: now,
	}, nil
}

func (s *Service) ListUptimeMonitors(ctx context.Context, projectID string) ([]UptimeMonitor, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, project_id, name, url, check_type, interval_secs, keyword,
       status, enabled, uptime_pct, latency_ms, last_checked, created_at
FROM observe_uptime_monitors WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UptimeMonitor
	for rows.Next() {
		var m UptimeMonitor
		if err := rows.Scan(
			&m.ID, &m.ProjectID, &m.Name, &m.URL, &m.CheckType, &m.IntervalSecs,
			&m.Keyword, &m.Status, &m.Enabled, &m.UptimePct, &m.LatencyMs,
			&m.LastChecked, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		m.History = s.uptimeHistory(ctx, m.ID)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) uptimeHistory(ctx context.Context, monitorID string) []string {
	rows, err := s.db.QueryContext(ctx,
		`SELECT status FROM observe_uptime_checks
         WHERE monitor_id = $1 ORDER BY checked_at DESC LIMIT 90`, monitorID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var st string
		if rows.Scan(&st) == nil {
			out = append(out, st)
		}
	}
	return out
}

func (s *Service) DeleteUptimeMonitor(ctx context.Context, projectID, monitorID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM observe_uptime_monitors WHERE id = $1 AND project_id = $2`,
		monitorID, projectID)
	return err
}

// RunUptimeCheck performs an HTTP check for a monitor and persists the result.
func (s *Service) RunUptimeCheck(ctx context.Context, m *UptimeMonitor) {
	start := time.Now()
	status := "up"
	latency := 0
	errMsg := ""

	c := &http.Client{Timeout: 10 * time.Second}
	resp, err := c.Get(m.URL)
	latency = int(time.Since(start).Milliseconds())
	if err != nil {
		status = "down"
		errMsg = err.Error()
	} else {
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			status = "down"
			errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		} else if latency > 3000 {
			status = "degraded"
		}
	}

	checkID := uid.New("unique()")
	_, _ = s.db.ExecContext(ctx, `
INSERT INTO observe_uptime_checks (id, monitor_id, status, latency_ms, error_msg)
VALUES ($1,$2,$3,$4,$5)`, checkID, m.ID, status, latency, errMsg)

	// Recompute uptime % from last 90 checks
	var upTotal, upCount int
	rows, _ := s.db.QueryContext(ctx,
		`SELECT status FROM observe_uptime_checks WHERE monitor_id = $1 ORDER BY checked_at DESC LIMIT 90`, m.ID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var st string
			if rows.Scan(&st) == nil {
				upTotal++
				if st == "up" {
					upCount++
				}
			}
		}
	}
	pct := 100.0
	if upTotal > 0 {
		pct = float64(upCount) / float64(upTotal) * 100
	}
	_, _ = s.db.ExecContext(ctx, `
UPDATE observe_uptime_monitors
SET status = $1, latency_ms = $2, uptime_pct = $3, last_checked = NOW()
WHERE id = $4`, status, latency, pct, m.ID)
}

// ── Cron Monitors ─────────────────────────────────────────────────────────────

func (s *Service) CreateCronMonitor(ctx context.Context, projectID string, req CreateCronMonitorRequest) (*CronMonitor, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()
	tz := req.Timezone
	if tz == "" {
		tz = "UTC"
	}
	gp := req.GracePeriod
	if gp <= 0 {
		gp = 5
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO observe_cron_monitors (id, project_id, name, schedule, timezone, grace_period, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		id, projectID, req.Name, req.Schedule, tz, gp, now,
	)
	if err != nil {
		return nil, fmt.Errorf("observe: create cron monitor: %w", err)
	}
	return &CronMonitor{
		ID: id, ProjectID: projectID, Name: req.Name,
		Schedule: req.Schedule, Timezone: tz, GracePeriod: gp,
		Status: "waiting", Enabled: true, CreatedAt: now,
	}, nil
}

func (s *Service) ListCronMonitors(ctx context.Context, projectID string) ([]CronMonitor, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, project_id, name, schedule, timezone, grace_period,
       status, enabled, last_duration_ms, last_run_at, next_run_at, created_at
FROM observe_cron_monitors WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CronMonitor
	for rows.Next() {
		var m CronMonitor
		if err := rows.Scan(
			&m.ID, &m.ProjectID, &m.Name, &m.Schedule, &m.Timezone, &m.GracePeriod,
			&m.Status, &m.Enabled, &m.LastDurationMs, &m.LastRunAt, &m.NextRunAt, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		m.History = s.cronHistory(ctx, m.ID)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) cronHistory(ctx context.Context, monitorID string) []string {
	rows, err := s.db.QueryContext(ctx,
		`SELECT status FROM observe_cron_checkins WHERE monitor_id = $1 ORDER BY checked_at DESC LIMIT 30`, monitorID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var st string
		if rows.Scan(&st) == nil {
			out = append(out, st)
		}
	}
	return out
}

func (s *Service) ToggleCronMonitor(ctx context.Context, projectID, monitorID string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE observe_cron_monitors SET enabled = NOT enabled
WHERE id = $1 AND project_id = $2`, monitorID, projectID)
	return err
}

func (s *Service) DeleteCronMonitor(ctx context.Context, projectID, monitorID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM observe_cron_monitors WHERE id = $1 AND project_id = $2`,
		monitorID, projectID)
	return err
}

// CronCheckin records a check-in event from a running job.
func (s *Service) CronCheckin(ctx context.Context, monitorID string, req CronCheckinRequest) error {
	id := uid.New("unique()")
	status := req.Status
	if status == "" {
		status = "ok"
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO observe_cron_checkins (id, monitor_id, status, duration_ms, error_msg)
VALUES ($1,$2,$3,$4,$5)`,
		id, monitorID, status, req.DurationMs, req.ErrorMsg)
	if err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `
UPDATE observe_cron_monitors
SET status = $1, last_run_at = NOW(), last_duration_ms = $2
WHERE id = $3`, status, req.DurationMs, monitorID)
	return nil
}

// ── Alert Rules ───────────────────────────────────────────────────────────────

func (s *Service) CreateAlertRule(ctx context.Context, projectID string, req CreateAlertRuleRequest) (*AlertRule, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO observe_alert_rules
    (id, project_id, name, metric, operator, threshold, time_window, severity, channel, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		id, projectID, req.Name, req.Metric, req.Operator,
		req.Threshold, req.Window, req.Severity, req.Channel, now,
	)
	if err != nil {
		return nil, fmt.Errorf("observe: create alert rule: %w", err)
	}
	return &AlertRule{
		ID: id, ProjectID: projectID, Name: req.Name,
		Metric: req.Metric, Operator: req.Operator, Threshold: req.Threshold,
		Window: req.Window, Severity: req.Severity, Channel: req.Channel,
		Enabled: true, CreatedAt: now,
	}, nil
}

func (s *Service) ListAlerts(ctx context.Context, projectID string) (map[string]any, error) {
	// Rules
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, metric, operator, threshold, time_window, severity, channel, enabled, last_fired, created_at
FROM observe_alert_rules WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []AlertRule
	for rows.Next() {
		var r AlertRule
		r.ProjectID = projectID
		if err := rows.Scan(
			&r.ID, &r.Name, &r.Metric, &r.Operator, &r.Threshold, &r.Window,
			&r.Severity, &r.Channel, &r.Enabled, &r.LastFired, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}

	// Active incidents
	incRows, err := s.db.QueryContext(ctx, `
SELECT id, rule_id, project_id, rule_name, severity, value, fired_at, resolved_at
FROM observe_alert_incidents
WHERE project_id = $1 AND resolved_at IS NULL
ORDER BY fired_at DESC LIMIT 20`, projectID)
	if err != nil {
		return nil, err
	}
	defer incRows.Close()
	var incidents []AlertIncident
	for incRows.Next() {
		var inc AlertIncident
		if err := incRows.Scan(
			&inc.ID, &inc.RuleID, &inc.ProjectID, &inc.RuleName,
			&inc.Severity, &inc.Value, &inc.FiredAt, &inc.ResolvedAt,
		); err != nil {
			continue
		}
		incidents = append(incidents, inc)
	}

	return map[string]any{"rules": rules, "incidents": incidents}, nil
}

func (s *Service) ToggleAlertRule(ctx context.Context, projectID, ruleID string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE observe_alert_rules SET enabled = NOT enabled
WHERE id = $1 AND project_id = $2`, ruleID, projectID)
	return err
}

func (s *Service) DeleteAlertRule(ctx context.Context, projectID, ruleID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM observe_alert_rules WHERE id = $1 AND project_id = $2`,
		ruleID, projectID)
	return err
}

// ── Extra error helpers ───────────────────────────────────────────────────────

func (s *Service) SetErrorPriority(ctx context.Context, projectID, errorID, priority string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE observe_errors SET priority = $1 WHERE id = $2 AND project_id = $3`,
		priority, errorID, projectID)
	return err
}

func (s *Service) AssignError(ctx context.Context, projectID, errorID, assignee string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE observe_errors SET assignee = $1 WHERE id = $2 AND project_id = $3`,
		assignee, errorID, projectID)
	return err
}

func (s *Service) AddActivity(ctx context.Context, errorID, activityType, actorName, text string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO observe_error_activity (id, error_id, type, user_name, text) VALUES ($1,$2,$3,$4,$5)`,
		uid.New("unique()"), errorID, activityType, actorName, text)
	return err
}

// ── GetRelease ────────────────────────────────────────────────────────────────

func (s *Service) GetRelease(ctx context.Context, projectID, releaseID string) (*Release, error) {
	var r Release
	var commitsB []byte
	err := s.db.QueryRowContext(ctx, `
SELECT id, project_id, version, environment, commit_count, commits,
       crash_free_sessions_pct, new_issues, regressed_issues, fixed_issues,
       created_at, deployed_at
FROM observe_releases WHERE id = $1 AND project_id = $2`, releaseID, projectID).Scan(
		&r.ID, &r.ProjectID, &r.Version, &r.Environment, &r.CommitCount,
		&commitsB, &r.CrashFreeSessionsPct, &r.NewIssues, &r.RegressedIssues,
		&r.FixedIssues, &r.CreatedAt, &r.DeployedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Commits = sliceJSONB(commitsB)
	return &r, nil
}

// ── Request types ─────────────────────────────────────────────────────────────

type CaptureErrorRequest struct {
	Title          string           `json:"title"`
	ErrorType      string           `json:"errorType"`
	Level          string           `json:"level"`
	StackTrace     string           `json:"stackTrace"`
	Breadcrumbs    []map[string]any `json:"breadcrumbs"`
	UserContext    map[string]any   `json:"userContext"`
	RequestContext map[string]any   `json:"requestContext"`
	RuntimeContext map[string]any   `json:"runtimeContext"`
	Tags           map[string]any   `json:"tags"`
	Environment    string           `json:"environment"`
	Release        string           `json:"release"`
}

type CaptureLogRequest struct {
	Level       string         `json:"level"`
	Message     string         `json:"message"`
	Source      string         `json:"source"`
	Environment string         `json:"environment"`
	Release     string         `json:"release"`
	Meta        map[string]any `json:"meta"`
	TraceID     string         `json:"traceId"`
	SpanID      string         `json:"spanId"`
}

type RecordPerfRequest struct {
	Method   string  `json:"method"`
	Path     string  `json:"path"`
	P50Ms    float64 `json:"p50Ms"`
	P75Ms    float64 `json:"p75Ms"`
	P95Ms    float64 `json:"p95Ms"`
	P99Ms    float64 `json:"p99Ms"`
	RPS      float64 `json:"rps"`
	ErrorPct float64 `json:"errorPct"`
	ReqCount int64   `json:"reqCount"`
}

type WebVitalsRequest struct {
	PageURL string   `json:"pageUrl"`
	LCP     *float64 `json:"lcp"`
	FID     *float64 `json:"fid"`
	CLS     *float64 `json:"cls"`
	TTFB    *float64 `json:"ttfb"`
	FCP     *float64 `json:"fcp"`
	INP     *float64 `json:"inp"`
}

type CreateReleaseRequest struct {
	Version     string           `json:"version"`
	Environment string           `json:"environment"`
	Commits     []map[string]any `json:"commits"`
}

type CreateReplayRequest struct {
	SessionID    string           `json:"sessionId"`
	UserID       string           `json:"userId"`
	User         string           `json:"user"`
	URL          string           `json:"url"`
	Browser      string           `json:"browser"`
	OS           string           `json:"os"`
	Country      string           `json:"country"`
	DurationSecs int              `json:"durationSecs"`
	ErrorCount   int              `json:"errorCount"`
	HasRageClick bool             `json:"hasRageClick"`
	HasDeadClick bool             `json:"hasDeadClick"`
	Events       []map[string]any `json:"events"`
	Network      []map[string]any `json:"network"`
	Console      []map[string]any `json:"console"`
}

type CreateUptimeMonitorRequest struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	CheckType    string `json:"checkType"`
	IntervalSecs int    `json:"intervalSecs"`
	Keyword      string `json:"keyword"`
}

type CreateCronMonitorRequest struct {
	Name        string `json:"name"`
	Schedule    string `json:"schedule"`
	Timezone    string `json:"timezone"`
	GracePeriod int    `json:"gracePeriod"`
}

type CronCheckinRequest struct {
	Status     string `json:"status"`
	DurationMs *int   `json:"durationMs"`
	ErrorMsg   string `json:"errorMsg"`
}

type CreateAlertRuleRequest struct {
	Name      string  `json:"name"`
	Metric    string  `json:"metric"`
	Operator  string  `json:"operator"`
	Threshold float64 `json:"threshold"`
	Window    string  `json:"window"`
	Severity  string  `json:"severity"`
	Channel   string  `json:"channel"`
	Enabled   bool    `json:"enabled"`
}

// ── Uptime worker ─────────────────────────────────────────────────────────────

// StartUptimeWorker runs a background goroutine that polls all enabled uptime
// monitors at their configured interval. Call once from the router initialiser.
func (s *Service) StartUptimeWorker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runUptimeChecks(ctx)
			}
		}
	}()
}

func (s *Service) runUptimeChecks(ctx context.Context) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, project_id, name, url, check_type, interval_secs, keyword,
       status, enabled, uptime_pct, latency_ms, last_checked, created_at
FROM observe_uptime_monitors WHERE enabled = TRUE`)
	if err != nil {
		return
	}
	defer rows.Close()

	now := time.Now()
	for rows.Next() {
		var m UptimeMonitor
		var lastChecked sql.NullTime
		if err := rows.Scan(
			&m.ID, &m.ProjectID, &m.Name, &m.URL, &m.CheckType, &m.IntervalSecs,
			&m.Keyword, &m.Status, &m.Enabled, &m.UptimePct, &m.LatencyMs,
			&lastChecked, &m.CreatedAt,
		); err != nil {
			continue
		}
		interval := time.Duration(m.IntervalSecs) * time.Second
		if lastChecked.Valid && now.Sub(lastChecked.Time) < interval {
			continue // not due yet
		}
		// Run in goroutine to avoid blocking the ticker
		mon := m
		go s.RunUptimeCheck(ctx, &mon)
	}
}

// ignoredFields keeps the linter happy when strings package is imported
var _ = strings.TrimSpace
