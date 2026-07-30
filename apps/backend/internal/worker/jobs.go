package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/jobs"
	"github.com/mittolabs/applad/internal/metrics"
	"github.com/mittolabs/applad/internal/netguard"
	"github.com/redis/go-redis/v9"
)

// Jobs is the push-delivery dispatcher for the jobs queues. Queues created with
// a worker URL are pull-only until this worker runs them: it dequeues due jobs
// and POSTs each to the queue's worker URL, acking on a 2xx and retrying (with
// the queue's retry delay) or dead-lettering otherwise. Queues without a worker
// URL stay pull-only and are ignored here.
type Jobs struct {
	cfg    *config.Config
	db     *db.DB
	svc    *jobs.Service
	client *http.Client
}

func NewJobs(cfg *config.Config) *Jobs {
	return &Jobs{cfg: cfg}
}

func (w *Jobs) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	StartRedisHeartbeat(ctx, rdb, "jobs")
	// Touch the file heartbeat before the first tick so the compose healthcheck
	// does not see the worker as hung during start-up.
	Heartbeat()

	database, err := db.Connect(w.cfg.DatabaseDSN, w.cfg.DBMaxOpenConns, w.cfg.DBMaxIdleConns)
	if err != nil {
		return err
	}
	w.db = database
	w.svc = jobs.NewService(database)
	// The worker URL is user-controlled, so deliveries go through netguard to
	// keep a queue from being pointed at cloud metadata or an internal service.
	w.client = netguard.Client(30 * time.Second)

	slog.Info("jobs worker: dispatching push queues")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("jobs worker: shutting down")
			return nil
		case <-ticker.C:
			w.tick(ctx)
			Heartbeat()
		}
	}
}

func (w *Jobs) tick(ctx context.Context) {
	queues, err := w.svc.PushQueues(ctx)
	if err != nil {
		slog.Error("jobs worker: list push queues failed", "error", err)
		return
	}
	for _, q := range queues {
		if ctx.Err() != nil {
			return
		}
		w.drain(ctx, q)
	}
}

// drain empties one queue's due jobs, running up to Concurrency deliveries in
// parallel. Each goroutine dequeues (FOR UPDATE SKIP LOCKED, so replicas and
// siblings never claim the same job) until the queue has nothing ready.
func (w *Jobs) drain(ctx context.Context, q *jobs.Queue) {
	conc := q.Concurrency
	if conc <= 0 {
		conc = 1
	}
	var wg sync.WaitGroup
	for i := 0; i < conc; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if ctx.Err() != nil {
					return
				}
				job, err := w.svc.Dequeue(ctx, q.ID)
				if err != nil {
					slog.Error("jobs worker: dequeue failed", "queue", q.ID, "error", err)
					return
				}
				if job == nil {
					return // nothing ready
				}
				w.dispatch(ctx, q, job)
				Heartbeat()
			}
		}()
	}
	wg.Wait()
}

// dispatch delivers one job to its queue's worker URL and records the outcome.
func (w *Jobs) dispatch(ctx context.Context, q *jobs.Queue, job *jobs.Job) {
	status, deliverErr := w.deliver(ctx, q.WorkerURL, job)

	// The queue's retry limit caps attempts; fall back to the job's own limit if
	// the queue leaves it unset.
	maxAttempts := q.RetryLimit
	if maxAttempts <= 0 {
		maxAttempts = job.MaxAttempts
	}

	switch decideDispatch(status, deliverErr, job.Attempts, maxAttempts) {
	case dispatchAck:
		if err := w.svc.Ack(ctx, job.ID); err != nil {
			slog.Error("jobs worker: ack failed", "job", job.ID, "error", err)
		}
		metrics.QueueJobs.Inc("jobs", "completed")
	case dispatchRetry:
		delay := time.Duration(q.RetryDelayS) * time.Second
		if err := w.svc.Retry(ctx, job.ID, dispatchError(status, deliverErr), delay); err != nil {
			slog.Error("jobs worker: retry failed", "job", job.ID, "error", err)
		}
		metrics.QueueJobs.Inc("jobs", "retried")
		slog.Warn("jobs worker: delivery failed, retrying",
			"job", job.ID, "queue", q.ID, "attempt", job.Attempts, "status", status)
	case dispatchFail:
		if err := w.svc.Fail(ctx, job, dispatchError(status, deliverErr), q.DeadLetterQueueID); err != nil {
			slog.Error("jobs worker: fail failed", "job", job.ID, "error", err)
		}
		metrics.QueueJobs.Inc("jobs", "failed")
		slog.Error("jobs worker: delivery exhausted retries",
			"job", job.ID, "queue", q.ID, "attempts", job.Attempts)
	}
}

// deliver POSTs the job to the worker URL and returns the HTTP status (0 on a
// transport error) plus any transport error. The response body is drained and
// discarded so the connection can be reused.
func (w *Jobs) deliver(ctx context.Context, url string, job *jobs.Job) (int, error) {
	body, err := json.Marshal(map[string]interface{}{
		"$id":      job.ID,
		"queueId":  job.QueueID,
		"name":     job.Name,
		"payload":  job.Payload,
		"attempts": job.Attempts,
	})
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Applad-Jobs/1.0")

	resp, err := w.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	return resp.StatusCode, nil
}

// dispatchError renders a human-readable last_error for a failed delivery.
func dispatchError(status int, deliverErr error) string {
	if deliverErr != nil {
		return deliverErr.Error()
	}
	return fmt.Sprintf("worker returned status %d", status)
}

// dispatchAction is what the dispatcher should do with a job after one delivery
// attempt. It is computed by decideDispatch, kept pure so the retry/dead-letter
// rules are testable without a database or an HTTP server.
type dispatchAction int

const (
	dispatchAck   dispatchAction = iota // delivered (2xx): mark completed
	dispatchRetry                       // failed but attempts remain: re-queue with delay
	dispatchFail                        // failed and no attempts remain: fail / dead-letter
)

// decideDispatch turns a delivery result into an action. attempts is the job's
// attempt count after the dequeue that ran this delivery (so it is 1 on the
// first try); maxAttempts is the cap. A 2xx with no transport error acks;
// anything else retries while attempts are below the cap and fails once they
// reach it.
func decideDispatch(status int, transportErr error, attempts, maxAttempts int) dispatchAction {
	if transportErr == nil && status >= 200 && status < 300 {
		return dispatchAck
	}
	if attempts < maxAttempts {
		return dispatchRetry
	}
	return dispatchFail
}
