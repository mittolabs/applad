// Package usage implements time-series usage analytics for projects.
package usage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// UsageMetric represents a single metric data point.
type UsageMetric struct {
	ID        string    `json:"$id"`
	ProjectID string    `json:"projectId"`
	Metric    string    `json:"metric"`
	Value     int64     `json:"value"`
	Period    string    `json:"period"`
	Timestamp time.Time `json:"timestamp"`
}

// TimeSeriesPoint represents a single point in a time-series response.
type TimeSeriesPoint struct {
	Timestamp string `json:"timestamp"`
	Value     int64  `json:"value"`
}

// ProjectStats holds aggregate project statistics.
type ProjectStats struct {
	ProjectID      string `json:"projectId"`
	TotalRequests  int64  `json:"totalRequests"`
	TotalUsers     int64  `json:"totalUsers"`
	TotalStorage   int64  `json:"totalStorage"`
	TotalExecutions int64 `json:"totalExecutions"`
}

// Service handles usage analytics business logic.
type Service struct {
	db *db.DB
}

// NewService creates a new usage Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Record inserts a metric row with the current hour timestamp.
func (s *Service) Record(ctx context.Context, projectID, metric string, value int64) error {
	id := uid.New("unique()")
	now := time.Now().UTC().Truncate(time.Hour)
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO usage_metrics (id, project_id, metric, value, period, timestamp) VALUES ($1, $2, $3, $4, $5, $6)",
		id, projectID, metric, value, "1h", now)
	if err != nil {
		return fmt.Errorf("usage: record: %w", err)
	}
	return nil
}

// GetUsage returns time-series data for a metric. Range: 24h, 7d, 30d.
// Groups by hour for 24h, by day for 7d and 30d.
func (s *Service) GetUsage(ctx context.Context, projectID, metric, rangeStr string) ([]TimeSeriesPoint, error) {
	var since time.Time
	var groupFormat string
	now := time.Now().UTC()

	switch rangeStr {
	case "24h":
		since = now.Add(-24 * time.Hour)
		groupFormat = `YYYY-MM-DD HH24":00:00"`
	case "7d":
		since = now.Add(-7 * 24 * time.Hour)
		groupFormat = "YYYY-MM-DD"
	case "30d":
		since = now.Add(-30 * 24 * time.Hour)
		groupFormat = "YYYY-MM-DD"
	default:
		since = now.Add(-24 * time.Hour)
		groupFormat = `YYYY-MM-DD HH24":00:00"`
	}

	query := fmt.Sprintf(
		"SELECT to_char(timestamp AT TIME ZONE 'UTC', '%s') AS period, SUM(value) AS total FROM usage_metrics WHERE project_id = $1 AND metric = $2 AND timestamp >= $3 GROUP BY period ORDER BY period ASC",
		groupFormat)

	rows, err := s.db.QueryContext(ctx, query, projectID, metric, since)
	if err != nil {
		return nil, fmt.Errorf("usage: get: %w", err)
	}
	defer rows.Close()

	var points []TimeSeriesPoint
	for rows.Next() {
		var p TimeSeriesPoint
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	if points == nil {
		points = []TimeSeriesPoint{}
	}
	return points, nil
}

// GetProjectStats returns aggregate stats for a project.
func (s *Service) GetProjectStats(ctx context.Context, projectID string) (*ProjectStats, error) {
	stats := &ProjectStats{ProjectID: projectID}

	// Total requests from usage_metrics
	s.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(value), 0) FROM usage_metrics WHERE project_id = $1 AND metric = 'requests'",
		projectID).Scan(&stats.TotalRequests) //nolint:errcheck

	// Total users
	row := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM users WHERE project_id = $1", projectID)
	if err := row.Scan(&stats.TotalUsers); err != nil && err != sql.ErrNoRows {
		stats.TotalUsers = 0
	}

	// Total storage bytes
	row = s.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(size), 0) FROM files WHERE project_id = $1", projectID)
	if err := row.Scan(&stats.TotalStorage); err != nil && err != sql.ErrNoRows {
		stats.TotalStorage = 0
	}

	// Total function executions
	row = s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM function_executions WHERE project_id = $1", projectID)
	if err := row.Scan(&stats.TotalExecutions); err != nil && err != sql.ErrNoRows {
		stats.TotalExecutions = 0
	}

	return stats, nil
}
