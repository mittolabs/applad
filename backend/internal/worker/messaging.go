package worker

import (
	"log"

	"github.com/mittolabs/applad/internal/config"
)

type Messaging struct {
	cfg  *config.Config
	stop chan struct{}
}

func NewMessaging(cfg *config.Config) *Messaging {
	return &Messaging{cfg: cfg, stop: make(chan struct{})}
}

func (w *Messaging) Start() error {
	log.Println("messaging worker: listening for jobs")
	<-w.stop
	return nil
}

func (w *Messaging) Stop() { close(w.stop) }
