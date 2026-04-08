package worker

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/redis/go-redis/v9"
)

// Cron fires deploy target executions on a schedule.
// It polls all targets with a non-empty cron expression once per minute.
type Cron struct {
	cfg   *config.Config
	stop  chan struct{}
	queue *queue.Queue
	db    *db.DB
}

func NewCron(cfg *config.Config) *Cron {
	return &Cron{cfg: cfg, stop: make(chan struct{})}
}

func (w *Cron) Start() error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	database, err := db.Connect(w.cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	w.db = database

	log.Println("cron worker: starting")

	// Wait until the top of the next minute, then tick every minute.
	now := time.Now()
	delay := time.Duration(60-now.Second())*time.Second - time.Duration(now.Nanosecond())
	select {
	case <-w.stop:
		return nil
	case <-time.After(delay):
	}

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// Fire once immediately on the aligned minute.
	w.tick(context.Background())

	for {
		select {
		case <-w.stop:
			return nil
		case t := <-ticker.C:
			w.tick(context.WithValue(context.Background(), struct{}{}, t))
		}
	}
}

func (w *Cron) tick(ctx context.Context) {
	rows, err := w.db.QueryContext(ctx,
		`SELECT id, project_id, cron FROM deploy_targets WHERE cron != '' AND cron IS NOT NULL`)
	if err != nil {
		log.Printf("cron worker: query error: %v", err)
		return
	}
	defer rows.Close()

	now := time.Now().UTC()
	for rows.Next() {
		var id, projectID, cronExpr string
		if err := rows.Scan(&id, &projectID, &cronExpr); err != nil {
			continue
		}
		if w.shouldRun(cronExpr, now) {
			log.Printf("cron worker: firing target %s (expr=%q)", id, cronExpr)
			w.fireTarget(ctx, id, projectID)
		}
	}
}

// shouldRun returns true when the cron expression matches the given time (minute granularity).
// Supports standard 5-field cron: "minute hour day month weekday"
func (w *Cron) shouldRun(expr string, t time.Time) bool {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return false
	}
	return matchCronField(parts[0], t.Minute()) &&
		matchCronField(parts[1], t.Hour()) &&
		matchCronField(parts[2], t.Day()) &&
		matchCronField(parts[3], int(t.Month())) &&
		matchCronField(parts[4], int(t.Weekday()))
}

// matchCronField checks whether a single cron field matches a value.
// Supports: "*", "*/n" (step), and exact integer values.
func matchCronField(field string, value int) bool {
	if field == "*" {
		return true
	}
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err != nil || step <= 0 {
			return false
		}
		return value%step == 0
	}
	n, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return n == value
}

func (w *Cron) fireTarget(ctx context.Context, targetID, projectID string) {
	jobID := targetID + "-cron-" + time.Now().UTC().Format("20060102150405")
	if err := w.queue.Push(ctx, "builds", queue.Job{
		ID:   jobID,
		Type: "deploy_execution",
		Payload: map[string]interface{}{
			"targetId":  targetID,
			"projectId": projectID,
			"trigger":   "cron",
		},
	}); err != nil {
		log.Printf("cron worker: failed to push job for target %s: %v", targetID, err)
	}
}

func (w *Cron) Stop() { close(w.stop) }
