package worker

import (
	"log"

	"github.com/mittolabs/applad/internal/config"
)

type Databases struct {
	cfg  *config.Config
	stop chan struct{}
}

func NewDatabases(cfg *config.Config) *Databases {
	return &Databases{cfg: cfg, stop: make(chan struct{})}
}

func (w *Databases) Start() error {
	log.Println("databases worker: listening for jobs")
	<-w.stop
	return nil
}

func (w *Databases) Stop() { close(w.stop) }
