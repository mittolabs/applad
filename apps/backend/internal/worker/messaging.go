package worker

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/metrics"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

// Messaging processes queued messaging jobs (batch emails, notifications).
type Messaging struct {
	cfg   *config.Config
	queue *queue.Queue
}

func NewMessaging(cfg *config.Config) *Messaging {
	return &Messaging{cfg: cfg}
}

func (w *Messaging) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)
	StartRedisHeartbeat(ctx, rdb, "messaging")
	// Touch the file heartbeat before the first tick so the compose
	// healthcheck does not see the worker as hung during start-up.
	Heartbeat()
	w.queue.StartReaper(ctx, "messaging")

	slog.Info("messaging worker: listening for jobs")

	for {
		receipt, err := w.queue.Pop(ctx, "messaging")
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("messaging worker: shutting down")
				return nil
			}
			slog.Error("messaging worker: pop error", "error", err)
			continue
		}
		Heartbeat()

		if receipt == nil {
			continue
		}
		if processErr := w.process(ctx, receipt.Job); processErr != nil {
			metrics.QueueJobs.Inc("messaging", "failed")
			receipt.Nack()
		} else {
			metrics.QueueJobs.Inc("messaging", "completed")
			receipt.Ack()
		}
	}
}

func (w *Messaging) process(_ context.Context, job *queue.Job) error {
	switch job.Type {
	case "email":
		return w.sendEmail(job)
	default:
		slog.Warn("messaging worker: unknown job type", "type", job.Type, "job_id", job.ID)
		return nil
	}
}

func (w *Messaging) sendEmail(job *queue.Job) error {
	var recipients []string
	switch v := job.Payload["to"].(type) {
	case string:
		recipients = strings.Split(v, ",")
	case []interface{}:
		for _, r := range v {
			if s, ok := r.(string); ok {
				recipients = append(recipients, s)
			}
		}
	}
	subject, _ := job.Payload["subject"].(string)
	html, _ := job.Payload["html"].(string)

	if len(recipients) == 0 || subject == "" {
		slog.Warn("messaging worker: job missing recipients or subject", "job_id", job.ID)
		return nil
	}
	if w.cfg.SMTPHost == "" {
		slog.Warn("messaging worker: SMTP not configured, dropping job", "job_id", job.ID)
		return nil
	}

	addr := w.cfg.SMTPHost + ":" + w.cfg.SMTPPort
	var auth smtp.Auth
	if w.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", w.cfg.SMTPUser, w.cfg.SMTPPass, w.cfg.SMTPHost)
	}
	to := strings.Join(recipients, ", ")
	headers := []string{
		fmt.Sprintf("From: %s", w.cfg.SMTPFrom),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}
	msg := []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + html)

	if err := smtp.SendMail(addr, auth, w.cfg.SMTPFrom, recipients, msg); err != nil {
		slog.Error("messaging worker: send failed", "job_id", job.ID, "error", err)
		return err
	}
	slog.Info("messaging worker: sent", "job_id", job.ID, "recipients", len(recipients))
	return nil
}
