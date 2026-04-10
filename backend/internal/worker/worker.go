// Package worker provides the base interface and implementations for all
// Applad background worker processes. Each worker type is an independent
// process that consumes jobs from a dedicated Redis queue.
package worker

import (
	"context"
	"os"
	"time"
)

// Worker is the common interface for all Applad background workers.
type Worker interface {
	// Start runs the worker until ctx is cancelled.
	Start(ctx context.Context) error
}

// heartbeatPath is the file touched on each job loop iteration.
// Kubernetes liveness probes check that this file is newer than 2 minutes.
const heartbeatPath = "/tmp/.worker_alive"

// Heartbeat updates the liveness heartbeat file.
// Workers should call this once per loop iteration (whether or not a job was found).
func Heartbeat() {
	now := time.Now()
	if err := os.WriteFile(heartbeatPath, []byte(now.Format(time.RFC3339)), 0o644); err != nil {
		// Non-fatal — Kubernetes will eventually restart if writes fail persistently.
		_ = err
	}
}
