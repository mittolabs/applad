package worker

import (
	"context"
	"log"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

// Migrations processes async migration and maintenance jobs: re-running
// embedded migrations, optimizing tables, and cleaning up expired sessions.
type Migrations struct {
	cfg   *config.Config
	stop  chan struct{}
	queue *queue.Queue
	db    *db.DB
}

func NewMigrations(cfg *config.Config) *Migrations {
	return &Migrations{cfg: cfg, stop: make(chan struct{})}
}

func (w *Migrations) Start() error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	database, err := db.Connect(w.cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	w.db = database

	log.Println("migrations worker: listening for jobs")

	ctx := context.Background()
	for {
		select {
		case <-w.stop:
			return nil
		default:
			job, err := w.queue.Pop(ctx, "migrations")
			if err != nil {
				log.Printf("migrations worker: pop error: %v", err)
				continue
			}
			if job == nil {
				continue
			}
			w.process(ctx, job)
		}
	}
}

func (w *Migrations) process(ctx context.Context, job *queue.Job) {
	log.Printf("migrations worker: processing job %s type=%s", job.ID, job.Type)

	switch job.Type {
	case "run_migrations":
		if err := w.db.Migrate(); err != nil {
			log.Printf("migrations worker: migration failed: %v", err)
			return
		}
		log.Printf("migrations worker: migrations completed successfully")

	case "optimize_tables":
		tables := []string{"files", "sessions", "users"}
		for _, table := range tables {
			_, err := w.db.ExecContext(ctx, "VACUUM ANALYZE "+table)
			if err != nil {
				log.Printf("migrations worker: vacuum %s failed: %v", table, err)
			} else {
				log.Printf("migrations worker: vacuumed table %s", table)
			}
		}

	case "cleanup_sessions":
		result, err := w.db.ExecContext(ctx,
			"DELETE FROM sessions WHERE expires_at < NOW()")
		if err != nil {
			log.Printf("migrations worker: session cleanup failed: %v", err)
			return
		}
		affected, _ := result.RowsAffected()
		log.Printf("migrations worker: cleaned up %d expired sessions", affected)

	default:
		log.Printf("migrations worker: unknown job type %q", job.Type)
	}
}

func (w *Migrations) Stop() { close(w.stop) }
