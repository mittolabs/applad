// Package jobs provides named job queues with priority, retry policies,
// dead-letter queues, delayed execution, and job dependency tracking.
package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/uid"
)

// Queue is a named job queue.
type Queue struct {
	ID                string    `json:"$id"`
	ProjectID         string    `json:"projectId"`
	Name              string    `json:"name"`
	WorkerURL         string    `json:"workerUrl,omitempty"`
	Concurrency       int       `json:"concurrency"`
	RetryLimit        int       `json:"retryLimit"`
	RetryDelayS       int       `json:"retryDelaySeconds"`
	DeadLetterQueueID string    `json:"deadLetterQueueId,omitempty"`
	Paused            bool      `json:"paused"`
	Stats             *QStats   `json:"stats,omitempty"`
	CreatedAt         time.Time `json:"$createdAt"`
	UpdatedAt         time.Time `json:"$updatedAt"`
}

// QStats holds live observability counters for a queue.
type QStats struct {
	Pending   int64 `json:"pending"`
	Running   int64 `json:"running"`
	Completed int64 `json:"completed"`
	Failed    int64 `json:"failed"`
}

// Job is a single unit of work placed in a queue.
type Job struct {
	ID          string                 `json:"$id"`
	QueueID     string                 `json:"queueId"`
	ProjectID   string                 `json:"projectId"`
	Name        string                 `json:"name"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	Status      string                 `json:"status"`
	Priority    int                    `json:"priority"`
	RunAt       time.Time              `json:"runAt"`
	Attempts    int                    `json:"attempts"`
	MaxAttempts int                    `json:"maxAttempts"`
	LastError   string                 `json:"lastError,omitempty"`
	DependsOn   []string               `json:"dependsOn,omitempty"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	CreatedAt   time.Time              `json:"$createdAt"`
	UpdatedAt   time.Time              `json:"$updatedAt"`
}

// Service manages queues and jobs.
type Service struct {
	db *db.DB
}

// NewService creates a new jobs Service.
func NewService(database *db.DB) *Service {
	return &Service{db: database}
}

// ── Queues ────────────────────────────────────────────────────────────────────

// CreateQueue creates a named queue.
func (s *Service) CreateQueue(ctx context.Context, projectID, name, workerURL string, concurrency, retryLimit, retryDelayS int, deadLetterQueueID string) (*Queue, error) {
	q := &Queue{
		ID: uid.New(""), ProjectID: projectID, Name: name,
		WorkerURL: workerURL, Concurrency: concurrency,
		RetryLimit: retryLimit, RetryDelayS: retryDelayS,
		DeadLetterQueueID: deadLetterQueueID,
		CreatedAt:         time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if q.Concurrency == 0 {
		q.Concurrency = 10
	}
	if q.RetryLimit == 0 {
		q.RetryLimit = 3
	}
	if q.RetryDelayS == 0 {
		q.RetryDelayS = 60
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO job_queues (id, project_id, name, worker_url, concurrency, retry_limit, retry_delay_s, dead_letter_queue_id, paused, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,0,$9,$10)`,
		q.ID, q.ProjectID, q.Name, nullStr(q.WorkerURL), q.Concurrency,
		q.RetryLimit, q.RetryDelayS, nullStr(q.DeadLetterQueueID),
		q.CreatedAt, q.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return nil, fmt.Errorf("jobs: queue %q already exists", name)
		}
		return nil, fmt.Errorf("jobs: create queue: %w", err)
	}
	return q, nil
}

// GetQueue fetches a queue by ID.
func (s *Service) GetQueue(ctx context.Context, queueID, projectID string) (*Queue, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, name, COALESCE(worker_url,''), concurrency, retry_limit, retry_delay_s,
		        COALESCE(dead_letter_queue_id,''), paused, created_at, updated_at
		 FROM job_queues WHERE id = $1 AND project_id = $2`, queueID, projectID)
	return scanQueue(row)
}

// ListQueues returns all queues for a project.
func (s *Service) ListQueues(ctx context.Context, projectID string) ([]*Queue, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, COALESCE(worker_url,''), concurrency, retry_limit, retry_delay_s,
		        COALESCE(dead_letter_queue_id,''), paused, created_at, updated_at
		 FROM job_queues WHERE project_id = $1 ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Queue
	for rows.Next() {
		q, err := scanQueue(rows)
		if err != nil {
			return nil, err
		}
		// Attach stats
		q.Stats, _ = s.queueStats(ctx, q.ID)
		out = append(out, q)
	}
	return out, nil
}

// UpdateQueue patches mutable queue fields.
func (s *Service) UpdateQueue(ctx context.Context, queueID, projectID string, paused *bool, concurrency, retryLimit *int, workerURL *string) (*Queue, error) {
	q, err := s.GetQueue(ctx, queueID, projectID)
	if err != nil {
		return nil, fmt.Errorf("jobs: queue not found")
	}
	if paused != nil {
		q.Paused = *paused
	}
	if concurrency != nil {
		q.Concurrency = *concurrency
	}
	if retryLimit != nil {
		q.RetryLimit = *retryLimit
	}
	if workerURL != nil {
		q.WorkerURL = *workerURL
	}
	_, err = s.db.ExecContext(ctx,
		"UPDATE job_queues SET worker_url=$1, concurrency=$2, retry_limit=$3, paused=$4, updated_at=$5 WHERE id=$6",
		nullStr(q.WorkerURL), q.Concurrency, q.RetryLimit, boolInt(q.Paused), time.Now().UTC(), q.ID)
	return q, err
}

// DeleteQueue deletes a queue and all its jobs.
func (s *Service) DeleteQueue(ctx context.Context, queueID, projectID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM job_queues WHERE id = $1 AND project_id = $2", queueID, projectID)
	return err
}

// ── Jobs ──────────────────────────────────────────────────────────────────────

// Enqueue places a job in a queue.
func (s *Service) Enqueue(ctx context.Context, projectID, queueID, name string, payload map[string]interface{}, priority int, runAt time.Time, maxAttempts int, dependsOn []string) (*Job, error) {
	if runAt.IsZero() {
		runAt = time.Now().UTC()
	}
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	j := &Job{
		ID: uid.New(""), QueueID: queueID, ProjectID: projectID,
		Name: name, Payload: payload, Status: "pending",
		Priority: priority, RunAt: runAt, MaxAttempts: maxAttempts,
		DependsOn: dependsOn,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	payloadJSON, _ := json.Marshal(payload)
	depsJSON, _ := json.Marshal(dependsOn)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO jobs (id, queue_id, project_id, name, payload, status, priority, run_at, attempts, max_attempts, depends_on, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,0,$9,$10,$11,$12)`,
		j.ID, j.QueueID, j.ProjectID, j.Name, nullBytes(payloadJSON), j.Status,
		j.Priority, j.RunAt, j.MaxAttempts, nullBytes(depsJSON),
		j.CreatedAt, j.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("jobs: enqueue: %w", err)
	}
	return j, nil
}

// GetJob fetches a job by ID.
func (s *Service) GetJob(ctx context.Context, jobID, projectID string) (*Job, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, queue_id, project_id, name, payload, status, priority, run_at, attempts, max_attempts,
		        COALESCE(last_error,''), depends_on, completed_at, created_at, updated_at
		 FROM jobs WHERE id = $1 AND project_id = $2`, jobID, projectID)
	return scanJob(row)
}

// ListJobs returns jobs for a queue with optional status filter.
func (s *Service) ListJobs(ctx context.Context, queueID, projectID, status string, limit, offset int) ([]*Job, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	n := 2
	where := "queue_id = $1 AND project_id = $2"
	args := []interface{}{queueID, projectID}
	if status != "" {
		n++
		where += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, status)
	}
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs WHERE "+where, countArgs...).Scan(&total) //nolint:errcheck
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT id, queue_id, project_id, name, payload, status, priority, run_at, attempts, max_attempts,
		        COALESCE(last_error,''), depends_on, completed_at, created_at, updated_at
		 FROM jobs WHERE %s ORDER BY priority DESC, run_at ASC LIMIT $%d OFFSET $%d`, where, n+1, n+2), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, j)
	}
	return out, total, nil
}

// CancelJob marks a pending job as cancelled.
func (s *Service) CancelJob(ctx context.Context, jobID, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE jobs SET status='cancelled', updated_at=$1 WHERE id=$2 AND project_id=$3 AND status='pending'",
		time.Now().UTC(), jobID, projectID)
	return err
}

// ── Workers call these ────────────────────────────────────────────────────────

// Dequeue claims the next due job from a queue (for worker use).
func (s *Service) Dequeue(ctx context.Context, queueID string) (*Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	row := tx.QueryRowContext(ctx,
		`SELECT id FROM jobs
		 WHERE queue_id = $1 AND status = 'pending' AND run_at <= $2 AND attempts < max_attempts
		 ORDER BY priority DESC, run_at ASC LIMIT 1 FOR UPDATE SKIP LOCKED`,
		queueID, time.Now().UTC())
	var jobID string
	if err := row.Scan(&jobID); err != nil {
		return nil, nil // no jobs ready
	}
	_, err = tx.ExecContext(ctx,
		"UPDATE jobs SET status='running', attempts=attempts+1, updated_at=$1 WHERE id=$2",
		time.Now().UTC(), jobID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	row2 := s.db.QueryRowContext(ctx,
		`SELECT id, queue_id, project_id, name, payload, status, priority, run_at, attempts, max_attempts,
		        COALESCE(last_error,''), depends_on, completed_at, created_at, updated_at
		 FROM jobs WHERE id = $1`, jobID)
	return scanJob(row2)
}

// Ack marks a job as completed.
func (s *Service) Ack(ctx context.Context, jobID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE jobs SET status='completed', completed_at=$1, updated_at=$2 WHERE id=$3",
		time.Now().UTC(), time.Now().UTC(), jobID)
	return err
}

// Nack records a failure and re-queues or moves to DLQ.
func (s *Service) Nack(ctx context.Context, jobID, errMsg string) error {
	row := s.db.QueryRowContext(ctx, "SELECT attempts, max_attempts, queue_id FROM jobs WHERE id=$1", jobID)
	var attempts, maxAttempts int
	var queueID string
	if err := row.Scan(&attempts, &maxAttempts, &queueID); err != nil {
		return err
	}

	status := "pending"
	if attempts >= maxAttempts {
		status = "failed"
	}
	_, err := s.db.ExecContext(ctx,
		"UPDATE jobs SET status=$1, last_error=$2, updated_at=$3 WHERE id=$4",
		status, errMsg, time.Now().UTC(), jobID)
	return err
}

// ── Dispatcher (push delivery) ──────────────────────────────────────────────────

// PushQueues returns every queue across all projects that has a worker URL set
// and is not paused. The jobs dispatcher worker calls this to discover which
// queues push their jobs to an HTTP endpoint rather than being pulled.
func (s *Service) PushQueues(ctx context.Context) ([]*Queue, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, name, COALESCE(worker_url,''), concurrency, retry_limit, retry_delay_s,
		        COALESCE(dead_letter_queue_id,''), paused, created_at, updated_at
		 FROM job_queues
		 WHERE worker_url IS NOT NULL AND worker_url != '' AND paused = 0
		 ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Queue
	for rows.Next() {
		q, err := scanQueue(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, nil
}

// Retry re-queues a job to run again after delay, recording the last error.
// The dispatcher calls this when a delivery failed but the job still has
// attempts left, so a retry honours the queue's retryDelaySeconds rather than
// hammering the endpoint immediately.
func (s *Service) Retry(ctx context.Context, jobID, errMsg string, delay time.Duration) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		"UPDATE jobs SET status='pending', run_at=$1, last_error=$2, updated_at=$3 WHERE id=$4",
		now.Add(delay), errMsg, now, jobID)
	return err
}

// Fail marks a job as permanently failed and, if the queue names a dead-letter
// queue, copies the job into it so a message that exhausted its retries is not
// silently lost.
func (s *Service) Fail(ctx context.Context, job *Job, errMsg, deadLetterQueueID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE jobs SET status='failed', last_error=$1, updated_at=$2 WHERE id=$3",
		errMsg, time.Now().UTC(), job.ID)
	if err != nil {
		return err
	}
	if deadLetterQueueID != "" {
		if _, dlqErr := s.Enqueue(ctx, job.ProjectID, deadLetterQueueID, job.Name,
			job.Payload, job.Priority, time.Now().UTC(), job.MaxAttempts, job.DependsOn); dlqErr != nil {
			return fmt.Errorf("jobs: dead-letter enqueue: %w", dlqErr)
		}
	}
	return nil
}

// ── Stats ─────────────────────────────────────────────────────────────────────

func (s *Service) queueStats(ctx context.Context, queueID string) (*QStats, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT status, COUNT(*) FROM jobs WHERE queue_id = $1 GROUP BY status", queueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stats := &QStats{}
	for rows.Next() {
		var st string
		var cnt int64
		if err := rows.Scan(&st, &cnt); err != nil {
			return nil, err
		}
		switch st {
		case "pending":
			stats.Pending = cnt
		case "running":
			stats.Running = cnt
		case "completed":
			stats.Completed = cnt
		case "failed":
			stats.Failed = cnt
		}
	}
	return stats, nil
}

// ── scanners ──────────────────────────────────────────────────────────────────

func scanQueue(row interface{ Scan(...interface{}) error }) (*Queue, error) {
	q := &Queue{}
	var pausedInt int
	if err := row.Scan(&q.ID, &q.ProjectID, &q.Name, &q.WorkerURL,
		&q.Concurrency, &q.RetryLimit, &q.RetryDelayS,
		&q.DeadLetterQueueID, &pausedInt, &q.CreatedAt, &q.UpdatedAt); err != nil {
		return nil, err
	}
	q.Paused = pausedInt == 1
	return q, nil
}

func scanJob(row interface{ Scan(...interface{}) error }) (*Job, error) {
	j := &Job{}
	var payloadRaw, depsRaw []byte
	var completedAt *time.Time
	if err := row.Scan(&j.ID, &j.QueueID, &j.ProjectID, &j.Name, &payloadRaw,
		&j.Status, &j.Priority, &j.RunAt, &j.Attempts, &j.MaxAttempts,
		&j.LastError, &depsRaw, &completedAt, &j.CreatedAt, &j.UpdatedAt); err != nil {
		return nil, err
	}
	if len(payloadRaw) > 0 {
		json.Unmarshal(payloadRaw, &j.Payload) //nolint:errcheck
	}
	if len(depsRaw) > 0 {
		json.Unmarshal(depsRaw, &j.DependsOn) //nolint:errcheck
	}
	j.CompletedAt = completedAt
	return j, nil
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

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
