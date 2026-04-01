// Package worker provides the base interface and implementations for all
// Applad background worker processes. Each worker type is an independent
// process that consumes jobs from a dedicated Redis queue.
package worker

// Worker is the common interface for all Applad background workers.
type Worker interface {
	Start() error
	Stop()
}
