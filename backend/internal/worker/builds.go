package worker

import (
	"log"

	"github.com/mittolabs/applad/internal/config"
)

type Builds struct {
	cfg  *config.Config
	stop chan struct{}
}

func NewBuilds(cfg *config.Config) *Builds {
	return &Builds{cfg: cfg, stop: make(chan struct{})}
}

func (w *Builds) Start() error {
	log.Println("builds worker: listening for jobs")
	<-w.stop
	return nil
}

func (w *Builds) Stop() { close(w.stop) }
