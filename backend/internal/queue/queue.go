// Package queue provides a Redis-backed job queue for background workers.
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Job represents a background job.
type Job struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt time.Time              `json:"createdAt"`
}

// Queue wraps Redis list-based job queues.
type Queue struct {
	client *redis.Client
}

// New creates a queue backed by the given Redis client.
func New(client *redis.Client) *Queue {
	return &Queue{client: client}
}

// Push enqueues a job to the named queue.
func (q *Queue) Push(ctx context.Context, queueName string, job Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("queue: marshal: %w", err)
	}
	return q.client.LPush(ctx, "applad:queue:"+queueName, data).Err()
}

// Pop blocks until a job is available on the named queue or the context is cancelled.
// Uses BRPOP with a 5-second timeout to allow periodic stop-channel checks.
func (q *Queue) Pop(ctx context.Context, queueName string) (*Job, error) {
	result, err := q.client.BRPop(ctx, 5*time.Second, "applad:queue:"+queueName).Result()
	if err == redis.Nil {
		return nil, nil // timeout, no job available
	}
	if err != nil {
		return nil, err
	}
	if len(result) < 2 {
		return nil, nil
	}

	var job Job
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return nil, fmt.Errorf("queue: unmarshal: %w", err)
	}
	return &job, nil
}

// Len returns the number of pending jobs in a queue.
func (q *Queue) Len(ctx context.Context, queueName string) (int64, error) {
	return q.client.LLen(ctx, "applad:queue:"+queueName).Result()
}
