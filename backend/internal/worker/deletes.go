package worker

import (
	"log"

	"github.com/mittolabs/applad/internal/config"
)

type Deletes struct {
	cfg  *config.Config
	stop chan struct{}
}

func NewDeletes(cfg *config.Config) *Deletes {
	return &Deletes{cfg: cfg, stop: make(chan struct{})}
}

func (w *Deletes) Start() error {
	log.Println("deletes worker: listening for jobs")
	<-w.stop
	return nil
}

func (w *Deletes) Stop() { close(w.stop) }
