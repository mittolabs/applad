package worker

import (
	"log"

	"github.com/mittolabs/applad/internal/config"
)

type Executions struct {
	cfg  *config.Config
	stop chan struct{}
}

func NewExecutions(cfg *config.Config) *Executions {
	return &Executions{cfg: cfg, stop: make(chan struct{})}
}

func (w *Executions) Start() error {
	log.Println("executions worker: listening for jobs")
	<-w.stop
	return nil
}

func (w *Executions) Stop() { close(w.stop) }
