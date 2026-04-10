// Package queue provides a Redis-backed job queue with at-least-once delivery.
//
// Jobs are moved atomically from the pending list to a per-queue processing
// list (LMOVE). The caller must call Receipt.Ack() on success or Receipt.Nack()
// on failure. Unacked jobs are automatically requeued by the Reaper goroutine
// after a configurable visibility timeout (default 5 minutes).
//
// Nacked jobs increment a retry counter. After MaxRetries attempts they are
// moved to the dead-letter queue (applad:queue:{name}:dlq).
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultMaxRetries        = 3
	defaultVisibilityTimeout = 5 * time.Minute
	reaperInterval           = 1 * time.Minute
)

// Job represents a background job.
type Job struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Payload    map[string]interface{} `json:"payload"`
	CreatedAt  time.Time              `json:"createdAt"`
	Retries    int                    `json:"retries,omitempty"`
	MaxRetries int                    `json:"maxRetries,omitempty"`
}

// Receipt is returned by Pop. Call Ack on success, Nack on failure.
type Receipt struct {
	Job  *Job
	raw  string
	q    *Queue
	name string
	once sync.Once
}

// Ack removes the job from the processing list. Call after successful processing.
func (r *Receipt) Ack() {
	r.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := r.q.client.LRem(ctx, r.q.key(r.name, "processing"), 1, r.raw).Err(); err != nil {
			slog.Warn("queue: ack failed", "queue", r.name, "job_id", r.Job.ID, "error", err)
		}
	})
}

// Nack increments the retry counter and requeues the job, or moves it to the
// dead-letter queue if MaxRetries is exhausted.
func (r *Receipt) Nack() {
	r.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Remove from processing list
		r.q.client.LRem(ctx, r.q.key(r.name, "processing"), 1, r.raw) //nolint:errcheck

		r.Job.Retries++
		maxRetries := r.Job.MaxRetries
		if maxRetries <= 0 {
			maxRetries = defaultMaxRetries
		}

		if r.Job.Retries >= maxRetries {
			// Move to dead-letter queue
			data, _ := json.Marshal(r.Job)
			r.q.client.LPush(ctx, r.q.key(r.name, "dlq"), data) //nolint:errcheck
			slog.Warn("queue: job moved to dlq",
				"queue", r.name, "job_id", r.Job.ID, "retries", r.Job.Retries)
			return
		}

		// Exponential backoff requeue
		backoff := time.Duration(1<<uint(r.Job.Retries)) * time.Second
		time.Sleep(backoff)

		data, _ := json.Marshal(r.Job)
		if err := r.q.client.LPush(ctx, r.q.key(r.name, ""), data).Err(); err != nil {
			slog.Error("queue: nack requeue failed",
				"queue", r.name, "job_id", r.Job.ID, "error", err)
		}
	})
}

// Queue wraps Redis list-based job queues.
type Queue struct {
	client *redis.Client
}

// New creates a queue backed by the given Redis client.
func New(client *redis.Client) *Queue {
	return &Queue{client: client}
}

func (q *Queue) key(queueName, suffix string) string {
	k := "applad:queue:" + queueName
	if suffix != "" {
		k += ":" + suffix
	}
	return k
}

// Push enqueues a job to the named queue.
func (q *Queue) Push(ctx context.Context, queueName string, job Job) error {
	if job.MaxRetries <= 0 {
		job.MaxRetries = defaultMaxRetries
	}
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("queue: marshal: %w", err)
	}
	return q.client.LPush(ctx, q.key(queueName, ""), data).Err()
}

// Pop blocks until a job is available on the named queue or ctx is cancelled.
// Returns (nil, nil) on timeout — the caller should loop.
// Returns (nil, ctx.Err()) when ctx is cancelled — the caller should exit.
// Returns a Receipt on success — the caller MUST call Ack() or Nack().
func (q *Queue) Pop(ctx context.Context, queueName string) (*Receipt, error) {
	// LMOVE atomically moves one job from the pending list to the processing list.
	// BLMOVE blocks up to 5 seconds before returning redis.Nil (timeout).
	result, err := q.client.BLMove(ctx,
		q.key(queueName, ""),
		q.key(queueName, "processing"),
		"RIGHT", "LEFT",
		5*time.Second,
	).Result()

	if err == redis.Nil {
		return nil, nil // timeout, no job available
	}
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}

	var job Job
	if err := json.Unmarshal([]byte(result), &job); err != nil {
		return nil, fmt.Errorf("queue: unmarshal: %w", err)
	}
	return &Receipt{Job: &job, raw: result, q: q, name: queueName}, nil
}

// Len returns the number of pending jobs in a queue.
func (q *Queue) Len(ctx context.Context, queueName string) (int64, error) {
	return q.client.LLen(ctx, q.key(queueName, "")).Result()
}

// DLQLen returns the number of jobs in the dead-letter queue.
func (q *Queue) DLQLen(ctx context.Context, queueName string) (int64, error) {
	return q.client.LLen(ctx, q.key(queueName, "dlq")).Result()
}

// StartReaper launches a goroutine that periodically scans the processing list
// and requeues jobs that have been stuck longer than visibilityTimeout.
// Call this once per queue name in each worker's Start method.
// The goroutine exits when ctx is cancelled.
func (q *Queue) StartReaper(ctx context.Context, queueName string) {
	go func() {
		ticker := time.NewTicker(reaperInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				q.reap(ctx, queueName)
			}
		}
	}()
}

func (q *Queue) reap(ctx context.Context, queueName string) {
	processingKey := q.key(queueName, "processing")
	items, err := q.client.LRange(ctx, processingKey, 0, -1).Result()
	if err != nil {
		return
	}
	for _, raw := range items {
		var job Job
		if err := json.Unmarshal([]byte(raw), &job); err != nil {
			continue
		}
		if time.Since(job.CreatedAt) < defaultVisibilityTimeout {
			continue
		}
		// Job has been in processing longer than the timeout — assume worker crashed.
		slog.Warn("queue: reaping stuck job",
			"queue", queueName, "job_id", job.ID,
			"age", time.Since(job.CreatedAt).Round(time.Second))
		r := &Receipt{Job: &job, raw: raw, q: q, name: queueName}
		r.Nack()
	}
}
