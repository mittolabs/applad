package worker

import (
	"log"

	"github.com/mittolabs/applad/internal/config"
)

type Mails struct {
	cfg  *config.Config
	stop chan struct{}
}

func NewMails(cfg *config.Config) *Mails {
	return &Mails{cfg: cfg, stop: make(chan struct{})}
}

func (w *Mails) Start() error {
	log.Println("mails worker: listening for jobs")
	<-w.stop
	return nil
}

func (w *Mails) Stop() { close(w.stop) }
