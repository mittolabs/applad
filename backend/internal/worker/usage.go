package worker

import (
	"log"

	"github.com/mittolabs/applad/internal/config"
)

type Usage struct {
	cfg  *config.Config
	stop chan struct{}
}

func NewUsage(cfg *config.Config) *Usage {
	return &Usage{cfg: cfg, stop: make(chan struct{})}
}

func (w *Usage) Start() error {
	log.Println("usage worker: listening for jobs")
	<-w.stop
	return nil
}

func (w *Usage) Stop() { close(w.stop) }
