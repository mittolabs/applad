package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/mittolabs/applad/internal/uid"
)

/*
Uptime and cron monitors.

These are the two pieces of the old Observe API that stayed on the platform when
error monitoring moved out to Bugslad. The split follows one rule: Applad reports
what the platform itself watches, and Bugslad reports what an app tells it. A
cron runs on Applad and an uptime monitor is polled by Applad, so a self-hoster
must not need a second product to learn whether their own scheduled job fired.
*/

// UptimeMonitor is an HTTP/TCP availability check owned by a project.
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

// CronMonitor watches a scheduled job and flags runs that never checked in.
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

// CreateUptimeMonitorRequest is the body of POST /analytics/uptime.
type CreateUptimeMonitorRequest struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	CheckType    string `json:"checkType"`
	IntervalSecs int    `json:"intervalSecs"`
	Keyword      string `json:"keyword"`
}

// CreateCronMonitorRequest is the body of POST /analytics/crons.
type CreateCronMonitorRequest struct {
	Name        string `json:"name"`
	Schedule    string `json:"schedule"`
	Timezone    string `json:"timezone"`
	GracePeriod int    `json:"gracePeriod"`
}

// CronCheckinRequest is the body a running job posts to report a run.
type CronCheckinRequest struct {
	Status     string `json:"status"`
	DurationMs *int   `json:"durationMs"`
	ErrorMsg   string `json:"errorMsg"`
}

// ── Uptime ────────────────────────────────────────────────────────────────────

// CreateUptimeMonitor registers a new availability check for a project.
func (s *Service) CreateUptimeMonitor(ctx context.Context, projectID string, req CreateUptimeMonitorRequest) (*UptimeMonitor, error) {
	id := uid.New("unique()")
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO analytics_uptime_monitors
    (id, project_id, name, url, check_type, interval_secs, keyword, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, projectID, req.Name, req.URL, req.CheckType,
		req.IntervalSecs, req.Keyword, now,
	)
	if err != nil {
		return nil, fmt.Errorf("analytics: create monitor: %w", err)
	}
	return &UptimeMonitor{
		ID: id, ProjectID: projectID, Name: req.Name, URL: req.URL,
		CheckType: req.CheckType, IntervalSecs: req.IntervalSecs,
		// Never checked yet: saying "up" here is a guess that happens to be
		// about a URL nobody has fetched.
		Status: "pending", Enabled: true, CreatedAt: now,
	}, nil
}

// ListUptimeMonitors returns a project's monitors, newest first.
func (s *Service) ListUptimeMonitors(ctx context.Context, projectID string) ([]UptimeMonitor, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, project_id, name, url, check_type, interval_secs, keyword,
       status, enabled, uptime_pct, latency_ms, last_checked, created_at
FROM analytics_uptime_monitors WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
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
		`SELECT status FROM analytics_uptime_checks
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

// DeleteUptimeMonitor removes a monitor and, by cascade, its check history.
func (s *Service) DeleteUptimeMonitor(ctx context.Context, projectID, monitorID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM analytics_uptime_monitors WHERE id = $1 AND project_id = $2`,
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
INSERT INTO analytics_uptime_checks (id, monitor_id, status, latency_ms, error_msg)
VALUES ($1,$2,$3,$4,$5)`, checkID, m.ID, status, latency, errMsg)

	// Recompute uptime % from last 90 checks
	var upTotal, upCount int
	rows, _ := s.db.QueryContext(ctx,
		`SELECT status FROM analytics_uptime_checks WHERE monitor_id = $1 ORDER BY checked_at DESC LIMIT 90`, m.ID)
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
UPDATE analytics_uptime_monitors
SET status = $1, latency_ms = $2, uptime_pct = $3, last_checked = NOW()
WHERE id = $4`, status, latency, pct, m.ID)
}

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
FROM analytics_uptime_monitors WHERE enabled = TRUE`)
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

// ── Cron monitors ─────────────────────────────────────────────────────────────

// CreateCronMonitor registers a scheduled job to watch for missed runs.
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
INSERT INTO analytics_cron_monitors (id, project_id, name, schedule, timezone, grace_period, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		id, projectID, req.Name, req.Schedule, tz, gp, now,
	)
	if err != nil {
		return nil, fmt.Errorf("analytics: create cron monitor: %w", err)
	}
	return &CronMonitor{
		ID: id, ProjectID: projectID, Name: req.Name,
		Schedule: req.Schedule, Timezone: tz, GracePeriod: gp,
		Status: "waiting", Enabled: true, CreatedAt: now,
	}, nil
}

// ListCronMonitors returns a project's cron monitors, newest first.
func (s *Service) ListCronMonitors(ctx context.Context, projectID string) ([]CronMonitor, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, project_id, name, schedule, timezone, grace_period,
       status, enabled, last_duration_ms, last_run_at, next_run_at, created_at
FROM analytics_cron_monitors WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
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
		`SELECT status FROM analytics_cron_checkins WHERE monitor_id = $1 ORDER BY checked_at DESC LIMIT 30`, monitorID)
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

// ToggleCronMonitor flips a monitor between enabled and paused.
func (s *Service) ToggleCronMonitor(ctx context.Context, projectID, monitorID string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE analytics_cron_monitors SET enabled = NOT enabled
WHERE id = $1 AND project_id = $2`, monitorID, projectID)
	return err
}

// DeleteCronMonitor removes a monitor and, by cascade, its check-in history.
func (s *Service) DeleteCronMonitor(ctx context.Context, projectID, monitorID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM analytics_cron_monitors WHERE id = $1 AND project_id = $2`,
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
INSERT INTO analytics_cron_checkins (id, monitor_id, status, duration_ms, error_msg)
VALUES ($1,$2,$3,$4,$5)`,
		id, monitorID, status, req.DurationMs, req.ErrorMsg)
	if err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `
UPDATE analytics_cron_monitors
SET status = $1, last_run_at = NOW(), last_duration_ms = $2
WHERE id = $3`, status, req.DurationMs, monitorID)
	return nil
}
