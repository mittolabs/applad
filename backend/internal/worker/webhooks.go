package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

type Webhooks struct {
	cfg    *config.Config
	stop   chan struct{}
	queue  *queue.Queue
	client *http.Client
}

func NewWebhooks(cfg *config.Config) *Webhooks {
	return &Webhooks{cfg: cfg, stop: make(chan struct{})}
}

func (w *Webhooks) Start() error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)
	w.client = &http.Client{Timeout: 30 * time.Second}

	log.Println("webhooks worker: listening for jobs")

	ctx := context.Background()
	for {
		select {
		case <-w.stop:
			return nil
		default:
			job, err := w.queue.Pop(ctx, "webhooks")
			if err != nil {
				log.Printf("webhooks worker: pop error: %v", err)
				continue
			}
			if job == nil {
				continue
			}
			w.process(ctx, job)
		}
	}
}

func (w *Webhooks) process(ctx context.Context, job *queue.Job) {
	log.Printf("webhooks worker: processing job %s", job.ID)

	url, _ := job.Payload["url"].(string)
	event, _ := job.Payload["event"].(string)
	secret, _ := job.Payload["secret"].(string)

	if url == "" {
		log.Printf("webhooks worker: job %s missing url", job.ID)
		return
	}

	// Build webhook payload
	payload := map[string]interface{}{
		"event":     event,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      job.Payload["data"],
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("webhooks worker: marshal error: %v", err)
		return
	}

	// Retry up to 3 times with exponential backoff
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(backoff)
		}

		lastErr = w.deliver(ctx, url, secret, body)
		if lastErr == nil {
			log.Printf("webhooks worker: delivered job %s to %s", job.ID, url)
			return
		}
		log.Printf("webhooks worker: attempt %d failed for %s: %v", attempt+1, url, lastErr)
	}

	log.Printf("webhooks worker: giving up on job %s after 3 attempts: %v", job.ID, lastErr)
}

func (w *Webhooks) deliver(ctx context.Context, url, secret string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Applad-Webhooks/1.0")

	// Sign payload with HMAC-SHA256 if a secret is provided
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Applad-Signature", sig)
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

func (w *Webhooks) Stop() { close(w.stop) }
