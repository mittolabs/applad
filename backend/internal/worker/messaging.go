package worker

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

// Messaging processes queued messaging jobs (batch emails, notifications).
// Unlike the mails worker (transactional emails like password resets),
// this worker handles user-initiated messaging from the /messaging API.
type Messaging struct {
	cfg   *config.Config
	stop  chan struct{}
	queue *queue.Queue
}

func NewMessaging(cfg *config.Config) *Messaging {
	return &Messaging{cfg: cfg, stop: make(chan struct{})}
}

func (w *Messaging) Start() error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	log.Println("messaging worker: listening for jobs")

	ctx := context.Background()
	for {
		select {
		case <-w.stop:
			return nil
		default:
			job, err := w.queue.Pop(ctx, "messaging")
			if err != nil {
				log.Printf("messaging worker: pop error: %v", err)
				continue
			}
			if job == nil {
				continue
			}
			w.process(ctx, job)
		}
	}
}

func (w *Messaging) process(_ context.Context, job *queue.Job) {
	log.Printf("messaging worker: processing job %s type=%s", job.ID, job.Type)

	switch job.Type {
	case "email":
		w.sendEmail(job)
	default:
		log.Printf("messaging worker: unknown job type %q for job %s", job.Type, job.ID)
	}
}

func (w *Messaging) sendEmail(job *queue.Job) {
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
		log.Printf("messaging worker: job %s missing recipients or subject", job.ID)
		return
	}

	if w.cfg.SMTPHost == "" {
		log.Printf("messaging worker: SMTP not configured, dropping job %s", job.ID)
		return
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
		log.Printf("messaging worker: failed to send job %s: %v", job.ID, err)
		return
	}

	log.Printf("messaging worker: sent job %s to %d recipients", job.ID, len(recipients))
}

func (w *Messaging) Stop() { close(w.stop) }
