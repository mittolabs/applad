package worker

import (
	"log"

	"github.com/mittolabs/applad/internal/config"
)

type Migrations struct {
	cfg  *config.Config
	stop chan struct{}
}

func NewMigrations(cfg *config.Config) *Migrations {
	return &Migrations{cfg: cfg, stop: make(chan struct{})}
}

func (w *Migrations) Start() error {
	log.Println("migrations worker: listening for jobs")
	<-w.stop
	return nil
}

func (w *Migrations) Stop() { close(w.stop) }
