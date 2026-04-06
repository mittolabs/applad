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
		"UPDATE attributes SET status = 'processing' WHERE id = ?", attributeID)

	// Attribute creation is instant in our JSON-column model
	w.db.ExecContext(ctx,
		"UPDATE attributes SET status = 'available' WHERE id = ?", attributeID)

	log.Printf("databases worker: attribute %s is now available", attributeID)
}

func (w *Databases) processIndexCreate(ctx context.Context, job *queue.Job) {
	indexID, _ := job.Payload["indexId"].(string)
	if indexID == "" {
		return
	}

	w.db.ExecContext(ctx,
		"UPDATE `_indexes` SET status = 'building' WHERE id = ?", indexID)

	time.Sleep(500 * time.Millisecond)

	w.db.ExecContext(ctx,
		"UPDATE `_indexes` SET status = 'available' WHERE id = ?", indexID)

	log.Printf("databases worker: index %s is now available", indexID)
}

func (w *Databases) processCollectionStats(ctx context.Context, job *queue.Job) {
	collectionID, _ := job.Payload["tableId"].(string)
	if collectionID == "" {
		return
	}

	var count int
	err := w.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM documents WHERE collection_id = ?", collectionID).Scan(&count)
	if err != nil {
		log.Printf("databases worker: stats error for collection %s: %v", collectionID, err)
		return
	}

	log.Printf("databases worker: collection %s has %d documents", collectionID, count)
}

func (w *Databases) Stop() { close(w.stop) }
