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

type Mails struct {
	cfg   *config.Config
	stop  chan struct{}
	queue *queue.Queue
}

func NewMails(cfg *config.Config) *Mails {
	return &Mails{cfg: cfg, stop: make(chan struct{})}
}

func (w *Mails) Start() error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	log.Println("mails worker: listening for jobs")

	ctx := context.Background()
	for {
		select {
		case <-w.stop:
			return nil
		default:
			job, err := w.queue.Pop(ctx, "mails")
			if err != nil {
				log.Printf("mails worker: pop error: %v", err)
				continue
			}
			if job == nil {
				continue
			}
			w.process(ctx, job)
		}
	}
}

func (w *Mails) process(_ context.Context, job *queue.Job) {
	log.Printf("mails worker: processing job %s type=%s", job.ID, job.Type)

	to, _ := job.Payload["to"].(string)
	subject, _ := job.Payload["subject"].(string)
	html, _ := job.Payload["html"].(string)
	from, _ := job.Payload["from"].(string)

	if to == "" || subject == "" {
		log.Printf("mails worker: job %s missing to or subject", job.ID)
		return
	}
	if from == "" {
		from = w.cfg.SMTPFrom
	}

	if w.cfg.SMTPHost == "" {
		log.Printf("mails worker: SMTP not configured, dropping job %s", job.ID)
		return
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
		log.Printf("mails worker: failed to send job %s: %v", job.ID, err)
		return
	}

	log.Printf("mails worker: sent job %s to %s", job.ID, to)
}

func (w *Mails) Stop() { close(w.stop) }
