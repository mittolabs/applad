package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/worker"
)

func main() {
	cfg := config.Load()
	w := worker.NewMigrations(cfg)

	go func() {
		if err := w.Start(); err != nil {
			log.Fatalf("migrations worker error: %v", err)
		}
	}()

	log.Println("migrations worker started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	w.Stop()
	log.Println("migrations worker stopped")
}
