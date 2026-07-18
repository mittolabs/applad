package status

import (
	"context"
	"time"
)

// ComponentStatus is a component's current state plus rolling uptime.
type ComponentStatus struct {
	Key       string   `json:"key"`
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	LatencyMs int      `json:"latencyMs"`
	Uptime24h float64  `json:"uptime24h"`
	Uptime7d  float64  `json:"uptime7d"`
	Uptime90d float64  `json:"uptime90d"`
	History   []string `json:"history"` // per-day status, oldest→newest, up to 90 entries
}

// Incident is a status incident (auto-opened by the checker).
type Incident struct {
	ID         string     `json:"id"`
	Component  string     `json:"component"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	Severity   string     `json:"severity"`
	StartedAt  time.Time  `json:"startedAt"`
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`
}

// Snapshot is the full public status payload.
type Snapshot struct {
	Overall    string            `json:"overall"`
	UpdatedAt  time.Time         `json:"updatedAt"`
	Components []ComponentStatus `json:"components"`
	Incidents  []Incident        `json:"incidents"`
}

// Snapshot builds the public status payload from recorded checks and incidents.
func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	snap := Snapshot{Overall: statusOperational, UpdatedAt: time.Now().UTC()}

	worst := 0
	rank := map[string]int{statusOperational: 0, statusDegraded: 1, statusDown: 2}

	for _, c := range components {
		cs := ComponentStatus{Key: c.Key, Name: c.Name, Status: statusOperational}

		// Latest check drives current status + latency.
		var latest string
		err := s.db.QueryRowContext(ctx,
			`SELECT status, latency_ms FROM status_checks WHERE component=$1 ORDER BY checked_at DESC LIMIT 1`,
			c.Key,
		).Scan(&latest, &cs.LatencyMs)
		if err == nil {
			cs.Status = latest
		}

		cs.Uptime24h = s.uptime(ctx, c.Key, "24 hours")
		cs.Uptime7d = s.uptime(ctx, c.Key, "7 days")
		cs.Uptime90d = s.uptime(ctx, c.Key, "90 days")
		cs.History = s.history(ctx, c.Key)

		if rank[cs.Status] > worst {
			worst = rank[cs.Status]
		}
		snap.Components = append(snap.Components, cs)
	}

	for st, r := range rank {
		if r == worst {
			snap.Overall = st
		}
	}

	snap.Incidents = s.recentIncidents(ctx)
	return snap, nil
}

// uptime returns the percentage of operational checks over the given interval.
// 100 when there is no data yet (nothing has gone wrong).
func (s *Service) uptime(ctx context.Context, component, interval string) float64 {
	var pct *float64
	err := s.db.QueryRowContext(ctx,
		`SELECT count(*) FILTER (WHERE status='operational')::float / NULLIF(count(*),0) * 100
		   FROM status_checks
		  WHERE component=$1 AND checked_at > now() - $2::interval`,
		component, interval,
	).Scan(&pct)
	if err != nil || pct == nil {
		return 100
	}
	// Round to 2 decimals.
	return float64(int(*pct*100+0.5)) / 100
}

// history returns up to 90 per-day statuses (oldest→newest) for the uptime bar.
func (s *Service) history(ctx context.Context, component string) []string {
	rows, err := s.db.QueryContext(ctx,
		`SELECT date_trunc('day', checked_at) AS d,
		        bool_or(status='down')     AS any_down,
		        bool_or(status='degraded') AS any_deg
		   FROM status_checks
		  WHERE component=$1 AND checked_at > now() - interval '90 days'
		  GROUP BY d ORDER BY d ASC`,
		component,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var d time.Time
		var anyDown, anyDeg bool
		if err := rows.Scan(&d, &anyDown, &anyDeg); err != nil {
			continue
		}
		switch {
		case anyDown:
			out = append(out, statusDown)
		case anyDeg:
			out = append(out, statusDegraded)
		default:
			out = append(out, statusOperational)
		}
	}
	return out
}

func (s *Service) recentIncidents(ctx context.Context) []Incident {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, component, title, status, severity, started_at, resolved_at
		   FROM status_incidents
		  WHERE started_at > now() - interval '90 days'
		  ORDER BY started_at DESC LIMIT 20`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := []Incident{}
	for rows.Next() {
		var in Incident
		if err := rows.Scan(&in.ID, &in.Component, &in.Title, &in.Status, &in.Severity, &in.StartedAt, &in.ResolvedAt); err != nil {
			continue
		}
		out = append(out, in)
	}
	return out
}
