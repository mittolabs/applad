package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/metrics"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

type Webhooks struct {
	cfg    *config.Config
	queue  *queue.Queue
	client *http.Client
}

func NewWebhooks(cfg *config.Config) *Webhooks {
	return &Webhooks{cfg: cfg}
}

func (w *Webhooks) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)
	w.client = &http.Client{Timeout: 30 * time.Second}
	w.queue.StartReaper(ctx, "webhooks")

	slog.Info("webhooks worker: listening for jobs")

	for {
		receipt, err := w.queue.Pop(ctx, "webhooks")
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("webhooks worker: shutting down")
				return nil
			}
			slog.Error("webhooks worker: pop error", "error", err)
			continue
		}
		if receipt == nil {
			continue
		}
		Heartbeat()
		if processErr := w.process(ctx, receipt.Job); processErr != nil {
			metrics.QueueJobs.Inc("webhooks", "failed")
			receipt.Nack()
		} else {
			metrics.QueueJobs.Inc("webhooks", "completed")
			receipt.Ack()
		}
	}
}

func (w *Webhooks) process(ctx context.Context, job *queue.Job) error {
	url, _ := job.Payload["url"].(string)
	event, _ := job.Payload["event"].(string)
	secret, _ := job.Payload["secret"].(string)

	if url == "" {
		slog.Warn("webhooks worker: job missing url", "job_id", job.ID)
		return nil
	}

	payload := map[string]interface{}{
		"event":     event,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      job.Payload["data"],
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
		}
		lastErr = w.deliver(ctx, url, secret, body)
		if lastErr == nil {
			slog.Info("webhooks worker: delivered", "job_id", job.ID, "url", url)
			return nil
		}
		slog.Warn("webhooks worker: attempt failed", "attempt", attempt+1, "url", url, "error", lastErr)
	}
	return fmt.Errorf("all 3 attempts failed: %w", lastErr)
}

func (w *Webhooks) deliver(ctx context.Context, url, secret string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Applad-Webhooks/1.0")

	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		req.Header.Set("X-Applad-Signature", hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
