package worker

import (
	"log"

	"github.com/mittolabs/applad/internal/config"
)

type Certificates struct {
	cfg  *config.Config
	stop chan struct{}
}

func NewCertificates(cfg *config.Config) *Certificates {
	return &Certificates{cfg: cfg, stop: make(chan struct{})}
}

func (w *Certificates) Start() error {
	log.Println("certificates worker: listening for jobs")
	<-w.stop
	return nil
}

func (w *Certificates) Stop() { close(w.stop) }
