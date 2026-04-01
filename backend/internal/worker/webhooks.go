package worker

import (
	"log"

	"github.com/mittolabs/applad/internal/config"
)

type Webhooks struct {
	cfg  *config.Config
	stop chan struct{}
}

func NewWebhooks(cfg *config.Config) *Webhooks {
	return &Webhooks{cfg: cfg, stop: make(chan struct{})}
}

func (w *Webhooks) Start() error {
	log.Println("webhooks worker: listening for jobs")
	<-w.stop
	return nil
}

func (w *Webhooks) Stop() { close(w.stop) }
