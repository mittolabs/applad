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

type Mails struct {
	cfg   *config.Config
	queue *queue.Queue
}

func NewMails(cfg *config.Config) *Mails {
	return &Mails{cfg: cfg}
}

func (w *Mails) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)
	StartRedisHeartbeat(ctx, rdb, "mails")
	// Touch the file heartbeat before the first tick so the compose
	// healthcheck does not see the worker as hung during start-up.
	Heartbeat()
	w.queue.StartReaper(ctx, "mails")

	slog.Info("mails worker: listening for jobs")

	for {
		receipt, err := w.queue.Pop(ctx, "mails")
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("mails worker: shutting down")
				return nil
			}
			slog.Error("mails worker: pop error", "error", err)
			continue
		}
		Heartbeat()

		if receipt == nil {
			continue
		}
		if processErr := w.process(ctx, receipt.Job); processErr != nil {
			metrics.QueueJobs.Inc("mails", "failed")
			receipt.Nack()
		} else {
			metrics.QueueJobs.Inc("mails", "completed")
			receipt.Ack()
		}
	}
}

func (w *Mails) process(_ context.Context, job *queue.Job) error {
	to, _ := job.Payload["to"].(string)
	subject, _ := job.Payload["subject"].(string)
	html, _ := job.Payload["html"].(string)
	from, _ := job.Payload["from"].(string)

	if to == "" || subject == "" {
		slog.Warn("mails worker: job missing to or subject", "job_id", job.ID)
		return nil
	}
	if from == "" {
		from = w.cfg.SMTPFrom
	}
	if w.cfg.SMTPHost == "" {
		slog.Warn("mails worker: SMTP not configured, dropping job", "job_id", job.ID)
		return nil
	}

	addr := w.cfg.SMTPHost + ":" + w.cfg.SMTPPort
	var auth smtp.Auth
	if w.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", w.cfg.SMTPUser, w.cfg.SMTPPass, w.cfg.SMTPHost)
	}
	recipients := strings.Split(to, ",")
	headers := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}
	msg := []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + html)

	if err := smtp.SendMail(addr, auth, from, recipients, msg); err != nil {
		slog.Error("mails worker: send failed", "job_id", job.ID, "error", err)
		return err
	}
	slog.Info("mails worker: sent", "job_id", job.ID, "to", to)
	return nil
}
