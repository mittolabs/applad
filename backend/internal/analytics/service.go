// Package analytics provides first-party event tracking with funnels,
// retention curves, and session aggregation.
package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// Event is a single tracked analytics event.
type Event struct {
	ID         string                 `json:"$id"`
	ProjectID  string                 `json:"projectId"`
	UserID     string                 `json:"userId,omitempty"`
	SessionID  string                 `json:"sessionId,omitempty"`
	Event      string                 `json:"event"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	URL        string                 `json:"url,omitempty"`
	Referrer   string                 `json:"referrer,omitempty"`
	DeviceType string                 `json:"deviceType,omitempty"`
	Browser    string                 `json:"browser,omitempty"`
	Country    string                 `json:"country,omitempty"`
	CreatedAt  time.Time              `json:"$createdAt"`
}

// Session is an aggregated user session.
type Session struct {
	ID         string     `json:"$id"`
	ProjectID  string     `json:"projectId"`
	UserID     string     `json:"userId,omitempty"`
	DeviceType string     `json:"deviceType,omitempty"`
	Browser    string     `json:"browser,omitempty"`
	Country    string     `json:"country,omitempty"`
	EntryURL   string     `json:"entryUrl,omitempty"`
	ExitURL    string     `json:"exitUrl,omitempty"`
	PageViews  int        `json:"pageViews"`
	DurationS  int        `json:"durationSeconds"`
	StartedAt  time.Time  `json:"startedAt"`
	EndedAt    *time.Time `json:"endedAt,omitempty"`
}

// Funnel defines a multi-step conversion funnel.
type Funnel struct {
	ID        string    `json:"$id"`
	ProjectID string    `json:"projectId"`
	Name      string    `json:"name"`
	Steps     []string  `json:"steps"`
	CreatedAt time.Time `json:"$createdAt"`
	UpdatedAt time.Time `json:"$updatedAt"`
}

// FunnelResult holds conversion rates per step.
type FunnelResult struct {
	FunnelID string        `json:"funnelId"`
	Steps    []StepResult  `json:"steps"`
}

// StepResult is the conversion data for a single funnel step.
type StepResult struct {
	Step        string  `json:"step"`
	Count       int64   `json:"count"`
	Conversion  float64 `json:"conversionRate"` // % vs previous step
}

// Service handles analytics persistence and queries.
type Service struct {
	db *db.DB
}

// NewService creates a new analytics Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// Track records a single event.
func (s *Service) Track(ctx context.Context, e Event) (*Event, error) {
	if e.ProjectID == "" || e.Event == "" {
		return nil, fmt.Errorf("analytics: projectId and event are required")
	}
	e.ID = uid.New("")
	e.CreatedAt = time.Now().UTC()

	propsJSON, _ := json.Marshal(e.Properties)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO analytics_events
		 (id, project_id, user_id, session_id, event, properties, url, referrer, device_type, browser, country, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		e.ID, e.ProjectID, nullStr(e.UserID), nullStr(e.SessionID), e.Event,
		nullBytes(propsJSON), nullStr(e.URL), nullStr(e.Referrer),
		nullStr(e.DeviceType), nullStr(e.Browser), nullStr(e.Country), e.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("analytics: track: %w", err)
	}
	return &e, nil
}

// TrackBatch records multiple events in a single transaction.
func (s *Service) TrackBatch(ctx context.Context, events []Event) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint:errcheck
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO analytics_events
		 (id, project_id, user_id, session_id, event, properties, url, referrer, device_type, browser, country, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	count := 0
	for _, e := range events {
		if e.ProjectID == "" || e.Event == "" {
			continue
		}
		e.ID = uid.New("")
		e.CreatedAt = time.Now().UTC()
		propsJSON, _ := json.Marshal(e.Properties)
		_, err := stmt.ExecContext(ctx,
			e.ID, e.ProjectID, nullStr(e.UserID), nullStr(e.SessionID), e.Event,
			nullBytes(propsJSON), nullStr(e.URL), nullStr(e.Referrer),
			nullStr(e.DeviceType), nullStr(e.Browser), nullStr(e.Country), e.CreatedAt,
		)
		if err != nil {
			return 0, err
		}
		count++
	}
	return count, tx.Commit()
}

// QueryEvents returns events filtered by event name, user, and time range.
func (s *Service) QueryEvents(ctx context.Context, projectID, event, userID string, from, to time.Time, limit, offset int) ([]*Event, int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	where, args := eventsWhere(projectID, event, userID, from, to)
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM analytics_events WHERE "+where, countArgs...).Scan(&total) //nolint:errcheck

	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, project_id, COALESCE(user_id,''), COALESCE(session_id,''), event, properties, COALESCE(url,''), COALESCE(referrer,''), COALESCE(device_type,''), COALESCE(browser,''), COALESCE(country,''), created_at "+
			"FROM analytics_events WHERE "+where+" ORDER BY created_at DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*Event
	for rows.Next() {
		e := &Event{}
		var propsRaw []byte
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.UserID, &e.SessionID, &e.Event,
			&propsRaw, &e.URL, &e.Referrer, &e.DeviceType, &e.Browser, &e.Country, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		if len(propsRaw) > 0 {
			json.Unmarshal(propsRaw, &e.Properties) //nolint:errcheck
		}
		out = append(out, e)
	}
	return out, total, nil
}

// EventCounts returns counts grouped by event name for a time window.
func (s *Service) EventCounts(ctx context.Context, projectID string, from, to time.Time) (map[string]int64, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT event, COUNT(*) as cnt FROM analytics_events
		 WHERE project_id = ? AND created_at BETWEEN ? AND ?
		 GROUP BY event ORDER BY cnt DESC`,
		projectID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int64)
	for rows.Next() {
		var ev string
		var cnt int64
		if err := rows.Scan(&ev, &cnt); err != nil {
			return nil, err
		}
		out[ev] = cnt
	}
	return out, nil
}

// DAU returns daily active users for a date range.
func (s *Service) DAU(ctx context.Context, projectID string, from, to time.Time) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DATE(created_at) as day, COUNT(DISTINCT user_id) as dau
		 FROM analytics_events
		 WHERE project_id = ? AND user_id IS NOT NULL AND created_at BETWEEN ? AND ?
		 GROUP BY day ORDER BY day ASC`,
		projectID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var day time.Time
		var dau int64
		if err := rows.Scan(&day, &dau); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{"date": day.Format("2006-01-02"), "users": dau})
	}
	return out, nil
}

// ── Funnels ──────────────────────────────────────────────────────────────────

// CreateFunnel creates a new funnel definition.
func (s *Service) CreateFunnel(ctx context.Context, projectID, name string, steps []string) (*Funnel, error) {
	f := &Funnel{
		ID: uid.New(""), ProjectID: projectID, Name: name, Steps: steps,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	stepsJSON, _ := json.Marshal(steps)
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO analytics_funnels (id, project_id, name, steps, created_at, updated_at) VALUES (?,?,?,?,?,?)",
		f.ID, f.ProjectID, f.Name, stepsJSON, f.CreatedAt, f.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("analytics: create funnel: %w", err)
	}
	return f, nil
}

// ListFunnels returns all funnels for a project.
func (s *Service) ListFunnels(ctx context.Context, projectID string) ([]*Funnel, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, project_id, name, steps, created_at, updated_at FROM analytics_funnels WHERE project_id = ? ORDER BY created_at DESC",
		projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Funnel
	for rows.Next() {
		f := &Funnel{}
		var stepsRaw []byte
		if err := rows.Scan(&f.ID, &f.ProjectID, &f.Name, &stepsRaw, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(stepsRaw, &f.Steps) //nolint:errcheck
		out = append(out, f)
	}
	return out, nil
}

// DeleteFunnel deletes a funnel by ID.
func (s *Service) DeleteFunnel(ctx context.Context, funnelID, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM analytics_funnels WHERE id = ? AND project_id = ?", funnelID, projectID)
	return err
}

// AnalyzeFunnel computes step-by-step conversion for a funnel over a time window.
func (s *Service) AnalyzeFunnel(ctx context.Context, projectID, funnelID string, from, to time.Time) (*FunnelResult, error) {
	// Load funnel definition
	row := s.db.QueryRowContext(ctx, "SELECT id, steps FROM analytics_funnels WHERE id = ? AND project_id = ?", funnelID, projectID)
	f := &Funnel{}
	var stepsRaw []byte
	if err := row.Scan(&f.ID, &stepsRaw); err != nil {
		return nil, fmt.Errorf("analytics: funnel not found")
	}
	json.Unmarshal(stepsRaw, &f.Steps) //nolint:errcheck

	result := &FunnelResult{FunnelID: funnelID}
	var prev int64 = -1
	for _, step := range f.Steps {
		var cnt int64
		s.db.QueryRowContext(ctx,
			`SELECT COUNT(DISTINCT user_id) FROM analytics_events
			 WHERE project_id = ? AND event = ? AND created_at BETWEEN ? AND ?`,
			projectID, step, from, to,
		).Scan(&cnt) //nolint:errcheck

		sr := StepResult{Step: step, Count: cnt}
		if prev > 0 {
			sr.Conversion = float64(cnt) / float64(prev) * 100
		} else if prev == -1 {
			sr.Conversion = 100
		}
		result.Steps = append(result.Steps, sr)
		prev = cnt
	}
	return result, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func eventsWhere(projectID, event, userID string, from, to time.Time) (string, []interface{}) {
	where := "project_id = ?"
	args := []interface{}{projectID}
	if event != "" {
		where += " AND event = ?"
		args = append(args, event)
	}
	if userID != "" {
		where += " AND user_id = ?"
		args = append(args, userID)
	}
	if !from.IsZero() {
		where += " AND created_at >= ?"
		args = append(args, from)
	}
	if !to.IsZero() {
		where += " AND created_at <= ?"
		args = append(args, to)
	}
	return where, args
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}
