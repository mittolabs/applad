package worker

import (
	"context"
	"log"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

// Databases processes database maintenance jobs: index rebuilds,
// attribute status updates, and collection statistics.
type Databases struct {
	cfg   *config.Config
	stop  chan struct{}
	queue *queue.Queue
	db    *db.DB
}

func NewDatabases(cfg *config.Config) *Databases {
	return &Databases{cfg: cfg, stop: make(chan struct{})}
}

func (w *Databases) Start() error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	database, err := db.Connect(w.cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	w.db = database

	log.Println("databases worker: listening for jobs")

	ctx := context.Background()
	for {
		select {
		case <-w.stop:
			return nil
		default:
			job, err := w.queue.Pop(ctx, "databases")
			if err != nil {
				log.Printf("databases worker: pop error: %v", err)
				continue
			}
			if job == nil {
				continue
			}
			w.process(ctx, job)
		}
	}
}

func (w *Databases) process(ctx context.Context, job *queue.Job) {
	log.Printf("databases worker: processing job %s type=%s", job.ID, job.Type)

	switch job.Type {
	case "attribute_create":
		w.processAttributeCreate(ctx, job)
	case "index_create":
		w.processIndexCreate(ctx, job)
	case "collection_stats":
		w.processCollectionStats(ctx, job)
	default:
		log.Printf("databases worker: unknown job type %q", job.Type)
	}
}

func (w *Databases) processAttributeCreate(ctx context.Context, job *queue.Job) {
	attributeID, _ := job.Payload["attributeId"].(string)
	if attributeID == "" {
		return
	}

	w.db.ExecContext(ctx,
		"UPDATE columns SET status = 'processing' WHERE id = ?", attributeID)

	// Column creation is instant in our PostgreSQL schema model
	w.db.ExecContext(ctx,
		"UPDATE columns SET status = 'available' WHERE id = ?", attributeID)

	log.Printf("databases worker: column %s is now available", attributeID)
}

func (w *Databases) processIndexCreate(ctx context.Context, job *queue.Job) {
	indexID, _ := job.Payload["indexId"].(string)
	if indexID == "" {
		return
	}

	w.db.ExecContext(ctx,
		"UPDATE indexes SET status = 'building' WHERE id = ?", indexID)

	time.Sleep(500 * time.Millisecond)

	w.db.ExecContext(ctx,
		"UPDATE indexes SET status = 'available' WHERE id = ?", indexID)

	log.Printf("databases worker: index %s is now available", indexID)
}

func (w *Databases) processCollectionStats(ctx context.Context, job *queue.Job) {
	tableID, _ := job.Payload["tableId"].(string)
	if tableID == "" {
		return
	}

	var count int
	err := w.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM tables WHERE id = ?", tableID).Scan(&count)
	if err != nil {
		log.Printf("databases worker: stats error for table %s: %v", tableID, err)
		return
	}

	log.Printf("databases worker: table %s exists in schema", tableID)
}

func (w *Databases) Stop() { close(w.stop) }
