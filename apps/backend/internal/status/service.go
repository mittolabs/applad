// Package status implements Applad's self-monitoring: it periodically probes
// Applad's own components (API, database, cache, storage, workers), records the
// results, opens/resolves incidents, and exposes a public snapshot that powers
// the status page at status.applad.io. Applad monitoring itself.
package status

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/cache"
	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
	workernames "github.com/mittolabs/applad/internal/worker/names"
)

const (
	statusOperational = "operational"
	statusDegraded    = "degraded"
	statusDown        = "down"
)

// Component is one monitored piece of the platform.
type Component struct {
	Key  string
	Name string
}

// components is the ordered list shown on the status page.
var components = []Component{
	{Key: "api", Name: "API"},
	{Key: "postgres", Name: "Database"},
	{Key: "redis", Name: "Cache & queues"},
	{Key: "storage", Name: "Storage"},
	{Key: "workers", Name: "Background workers"},
}

// Service probes components and serves the aggregated status snapshot.
type Service struct {
	db    *db.DB
	cache *cache.Cache
	cfg   *config.Config
}

// NewService constructs a status Service sharing the API's db and cache.
func NewService(database *db.DB, cacheClient *cache.Cache, cfg *config.Config) *Service {
	return &Service{db: database, cache: cacheClient, cfg: cfg}
}

// Run probes every 30s until ctx is cancelled. Start it as a goroutine.
func (s *Service) Run(ctx context.Context) {
	s.runOnce(ctx)
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.runOnce(ctx)
		}
	}
}

type probe struct {
	status  string
	latency int
	errMsg  string
}

func (s *Service) runOnce(ctx context.Context) {
	for _, c := range components {
		p := s.probe(ctx, c.Key)
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO status_checks (id, component, status, latency_ms, error_msg) VALUES ($1,$2,$3,$4,$5)`,
			uid.New("unique()"), c.Key, p.status, p.latency, p.errMsg,
		); err != nil {
			slog.Error("status: record check failed", "component", c.Key, "error", err)
		}
		s.reconcileIncident(ctx, c, p)
	}
}

func (s *Service) probe(ctx context.Context, key string) probe {
	start := time.Now()
	ms := func() int { return int(time.Since(start).Milliseconds()) }
	switch key {
	case "api":
		// The checker runs inside the API process, so if this executes the API
		// is serving requests.
		return probe{status: statusOperational, latency: 0}
	case "postgres":
		if err := s.db.PingContext(ctx); err != nil {
			return probe{status: statusDown, errMsg: err.Error()}
		}
		return probe{status: statusOperational, latency: ms()}
	case "redis":
		if err := s.cache.Ping(ctx); err != nil {
			return probe{status: statusDown, errMsg: err.Error()}
		}
		return probe{status: statusOperational, latency: ms()}
	case "storage":
		if s.cfg.StorageDriver == "s3" {
			// Object storage is external; treat as operational (a real S3 HEAD
			// probe can be added later).
			return probe{status: statusOperational, latency: 0}
		}
		if s.cfg.StoragePath != "" {
			if _, err := os.Stat(s.cfg.StoragePath); err != nil {
				return probe{status: statusDegraded, errMsg: err.Error()}
			}
		}
		return probe{status: statusOperational, latency: ms()}
	case "workers":
		return s.probeWorkers(ctx)
	}
	return probe{status: statusOperational}
}

// workerHeartbeatTTL bounds how old a heartbeat value may be before its worker
// is considered stale. It matches the TTL StartRedisHeartbeat sets on the key,
// so a value older than this would already have expired; parsing it as well
// guards against a clock skew or a key that lingered past its worker's death.
const workerHeartbeatTTL = 90 * time.Second

// probeWorkers checks that every worker in the canonical roster is publishing a
// fresh status:worker:<name> heartbeat. Reporting "operational" merely because
// some key exists hid the common failure where one worker (cron) beats while
// the other ten are dead, so the whole expected set is verified: operational
// only when all are fresh, down when none are, degraded when some are missing
// or stale.
func (s *Service) probeWorkers(ctx context.Context) probe {
	expected := workernames.All
	keys := make([]string, len(expected))
	for i, name := range expected {
		keys[i] = "status:worker:" + name
	}
	vals, err := s.cache.Client().MGet(ctx, keys...).Result()
	if err != nil {
		return probe{status: statusDegraded, errMsg: err.Error()}
	}
	beats := make([]string, len(expected))
	for i, v := range vals {
		if str, ok := v.(string); ok {
			beats[i] = str
		}
	}
	return evaluateWorkerHeartbeats(expected, beats, time.Now().UTC())
}

// evaluateWorkerHeartbeats is the pure decision behind probeWorkers: given the
// expected worker names and each one's heartbeat value (empty when the key is
// missing/expired), it returns the aggregate probe. A value that is present but
// older than workerHeartbeatTTL counts as stale; an unparseable-but-present
// value is trusted, since the key existing at all means it is within its TTL.
func evaluateWorkerHeartbeats(expected, beats []string, now time.Time) probe {
	var missing []string
	for i, name := range expected {
		beat := ""
		if i < len(beats) {
			beat = beats[i]
		}
		if !heartbeatFresh(beat, now) {
			missing = append(missing, name)
		}
	}
	switch {
	case len(missing) == 0:
		return probe{status: statusOperational}
	case len(missing) == len(expected):
		return probe{status: statusDown, errMsg: "no worker heartbeats"}
	default:
		return probe{status: statusDegraded, errMsg: "workers not reporting: " + strings.Join(missing, ", ")}
	}
}

// heartbeatFresh reports whether a heartbeat value marks a live worker: present,
// and (when its RFC3339 timestamp parses) no older than workerHeartbeatTTL.
func heartbeatFresh(beat string, now time.Time) bool {
	if beat == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, beat)
	if err != nil {
		return true // present within its TTL; a format we do not recognise is still a beat
	}
	return now.Sub(t) <= workerHeartbeatTTL
}

// reconcileIncident opens an incident when a component first goes unhealthy and
// resolves it when the component recovers.
func (s *Service) reconcileIncident(ctx context.Context, c Component, p probe) {
	// Scoped to origin='auto': the checker manages only the incidents it opened.
	// A manually-posted incident is the operator's to resolve, never auto-closed.
	var openID string
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM status_incidents WHERE component=$1 AND status='investigating' AND origin='auto' ORDER BY started_at DESC LIMIT 1`,
		c.Key,
	).Scan(&openID)
	hasOpen := err == nil

	if p.status == statusOperational {
		if hasOpen {
			if _, err := s.db.ExecContext(ctx,
				`UPDATE status_incidents SET status='resolved', resolved_at=NOW() WHERE id=$1`, openID,
			); err != nil {
				slog.Error("status: resolve incident failed", "id", openID, "error", err)
			}
		}
		return
	}

	if !hasOpen {
		severity := "major"
		if p.status == statusDegraded {
			severity = "minor"
		}
		title := c.Name + " " + p.status
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO status_incidents (id, component, title, status, severity, origin) VALUES ($1,$2,$3,'investigating',$4,'auto')`,
			uid.New("unique()"), c.Key, title, severity,
		); err != nil {
			slog.Error("status: open incident failed", "component", c.Key, "error", err)
		}
	}
}
