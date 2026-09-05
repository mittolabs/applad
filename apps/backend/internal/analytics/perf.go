package analytics

import (
	"context"
	"database/sql"
	"time"

	"github.com/mittolabs/applad/internal/uid"
)

/*
Request performance and the Analytics overview.

The percentile snapshots here are measured by Applad itself, in the middleware
that wraps every project-scoped request (see PerfCollector). Client-reported
performance — web vitals, crash-free sessions — belongs with the errors it
explains, which is to say with Bugslad, not here.
*/

// PerfSnapshot is one hour of aggregated latency for a single route.
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

// RecordPerfRequest is one aggregated bucket, written by the collector.
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

// RecordPerf persists one hourly latency bucket for a route.
func (s *Service) RecordPerf(ctx context.Context, projectID string, req RecordPerfRequest) error {
	id := uid.New("unique()")
	bucket := time.Now().UTC().Truncate(time.Hour)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO analytics_perf_snapshots
    (id, project_id, method, path, p50_ms, p75_ms, p95_ms, p99_ms, rps, error_pct, req_count, bucket_hour)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT DO NOTHING`,
		id, projectID, req.Method, req.Path,
		req.P50Ms, req.P75Ms, req.P95Ms, req.P99Ms,
		req.RPS, req.ErrorPct, req.ReqCount, bucket,
	)
	return err
}

// GetPerformance returns per-route latency for the last 24h plus an hourly chart.
func (s *Service) GetPerformance(ctx context.Context, projectID string) (map[string]any, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT method, path,
       AVG(p50_ms), AVG(p75_ms), AVG(p95_ms), AVG(p99_ms),
       AVG(rps), AVG(error_pct), SUM(req_count)
FROM analytics_perf_snapshots
WHERE project_id = $1 AND bucket_hour >= NOW() - INTERVAL '24 hours'
GROUP BY method, path
ORDER BY AVG(p95_ms) DESC
LIMIT 50`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	endpoints := []map[string]any{}
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

	metrics := map[string]any{}
	if endpointCount > 0 {
		avgP50 := totalP50 / float64(endpointCount)
		metrics["p50Ms"] = avgP50
		metrics["p95Ms"] = totalP95 / float64(endpointCount)
		metrics["p99Ms"] = totalP99 / float64(endpointCount)
		metrics["apdex"] = apdexFor(avgP50)
	}

	// Hourly chart for last 24h
	chart := []map[string]any{}
	chartRows, _ := s.db.QueryContext(ctx, `
SELECT bucket_hour, AVG(p50_ms), AVG(p95_ms), AVG(p99_ms)
FROM analytics_perf_snapshots
WHERE project_id = $1 AND bucket_hour >= NOW() - INTERVAL '24 hours'
GROUP BY bucket_hour ORDER BY bucket_hour`, projectID)
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

	return map[string]any{
		"metrics":   metrics,
		"endpoints": endpoints,
		"chart":     chart,
	}, nil
}

// apdexFor scores a p50 latency: satisfied under T, tolerating under 4T.
func apdexFor(p50 float64) float64 {
	const T = 500.0
	switch {
	case p50 <= T:
		return 1.0
	case p50 <= 4*T:
		return 0.5
	default:
		return 0.0
	}
}

// GetOverview is the summary shown on the Analytics landing page.
//
// A statistic nobody has measured is left out rather than sent as its perfect
// value. An instance with no monitors reported 100% uptime and a 1.00 apdex,
// which are the two numbers people read as proof.
func (s *Service) GetOverview(ctx context.Context, projectID string) (map[string]any, error) {
	stats := map[string]any{}

	var eventsToday, activeUsers, activeSessions int64
	_ = s.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM analytics_events
WHERE project_id = $1 AND created_at >= NOW() - INTERVAL '24 hours'`, projectID).Scan(&eventsToday)
	_ = s.db.QueryRowContext(ctx, `
SELECT COUNT(DISTINCT user_id) FROM analytics_events
WHERE project_id = $1 AND user_id IS NOT NULL AND created_at >= NOW() - INTERVAL '24 hours'`, projectID).Scan(&activeUsers)
	_ = s.db.QueryRowContext(ctx, `
SELECT COUNT(DISTINCT session_id) FROM analytics_events
WHERE project_id = $1 AND session_id IS NOT NULL AND created_at >= NOW() - INTERVAL '24 hours'`, projectID).Scan(&activeSessions)

	stats["eventsToday"] = eventsToday
	stats["activeUsers"] = activeUsers
	stats["activeSessions"] = activeSessions

	var avgP50, avgP95 sql.NullFloat64
	_ = s.db.QueryRowContext(ctx, `
SELECT AVG(p50_ms), AVG(p95_ms) FROM analytics_perf_snapshots
WHERE project_id = $1 AND bucket_hour >= NOW() - INTERVAL '24 hours'`, projectID).Scan(&avgP50, &avgP95)
	if avgP95.Valid {
		stats["p95Ms"] = avgP95.Float64
	}
	if avgP50.Valid {
		stats["apdex"] = apdexFor(avgP50.Float64)
	}

	var avgUptime sql.NullFloat64
	_ = s.db.QueryRowContext(ctx, `
SELECT AVG(uptime_pct) FROM analytics_uptime_monitors
WHERE project_id = $1 AND enabled = TRUE`, projectID).Scan(&avgUptime)
	if avgUptime.Valid {
		stats["uptimePct"] = avgUptime.Float64
	}

	// Top events over the last 7 days, so the landing page has something to
	// show before anyone has built a funnel.
	topEvents := []map[string]any{}
	rows, _ := s.db.QueryContext(ctx, `
SELECT event, COUNT(*) AS cnt FROM analytics_events
WHERE project_id = $1 AND created_at >= NOW() - INTERVAL '7 days'
GROUP BY event ORDER BY cnt DESC LIMIT 8`, projectID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var ev string
			var cnt int64
			if rows.Scan(&ev, &cnt) == nil {
				topEvents = append(topEvents, map[string]any{"event": ev, "count": cnt})
			}
		}
	}

	to := time.Now().UTC()
	dau, err := s.DAU(ctx, projectID, to.AddDate(0, 0, -14), to)
	if err != nil || dau == nil {
		dau = []map[string]interface{}{}
	}

	return map[string]any{
		"stats":     stats,
		"topEvents": topEvents,
		"dau":       dau,
	}, nil
}
