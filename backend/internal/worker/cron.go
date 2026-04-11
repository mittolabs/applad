package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/mittolabs/applad/internal/config"
	"github.com/mittolabs/applad/internal/db"
	"github.com/mittolabs/applad/internal/queue"
	"github.com/mittolabs/applad/internal/workflows"
	"github.com/redis/go-redis/v9"
)

// Cron fires deploy target and function executions on a schedule.
type Cron struct {
	cfg   *config.Config
	queue *queue.Queue
	db    *db.DB
}

func NewCron(cfg *config.Config) *Cron {
	return &Cron{cfg: cfg}
}

func (w *Cron) Start(ctx context.Context) error {
	rdb := redis.NewClient(&redis.Options{Addr: w.cfg.RedisAddr})
	w.queue = queue.New(rdb)

	database, err := db.Connect(w.cfg.DatabaseDSN, w.cfg.DBMaxOpenConns, w.cfg.DBMaxIdleConns)
	if err != nil {
		return err
	}
	w.db = database

	slog.Info("cron worker: starting")

	// Wait until the top of the next minute, then tick every minute.
	now := time.Now()
	delay := time.Duration(60-now.Second())*time.Second - time.Duration(now.Nanosecond())
	select {
	case <-ctx.Done():
		return nil
	case <-time.After(delay):
	}

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	w.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("cron worker: shutting down")
			return nil
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *Cron) tick(ctx context.Context) {
	now := time.Now().UTC()

	rows, err := w.db.QueryContext(ctx,
		`SELECT id, project_id, cron FROM deploy_targets WHERE cron != '' AND cron IS NOT NULL`)
	if err != nil {
		slog.Error("cron worker: deploy query error", "error", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var id, projectID, cronExpr string
			if err := rows.Scan(&id, &projectID, &cronExpr); err != nil {
				continue
			}
			if w.shouldRun(cronExpr, now) {
				slog.Info("cron worker: firing deploy target", "id", id, "expr", cronExpr)
				w.fireTarget(ctx, id, projectID)
			}
		}
	}

	fnRows, err := w.db.QueryContext(ctx,
		`SELECT id, project_id, cron FROM functions WHERE cron IS NOT NULL AND cron != '' AND status = 'active'`)
	if err != nil {
		slog.Error("cron worker: functions query error", "error", err)
		return
	}
	defer fnRows.Close()
	for fnRows.Next() {
		var id, projectID, cronExpr string
		if err := fnRows.Scan(&id, &projectID, &cronExpr); err != nil {
			continue
		}
		if w.shouldRun(cronExpr, now) {
			slog.Info("cron worker: firing function", "id", id, "expr", cronExpr)
			w.fireFunction(ctx, id, projectID)
		}
	}

	wfRows, err := w.db.QueryContext(ctx,
		`SELECT id, project_id, trigger_config FROM workflows WHERE trigger_type = 'cron' AND status = 'active'`)
	if err != nil {
		slog.Error("cron worker: workflows query error", "error", err)
		return
	}
	defer wfRows.Close()
	svc := workflows.NewService(w.db, w.queue)
	for wfRows.Next() {
		var id, projectID string
		var tcJSON []byte
		if err := wfRows.Scan(&id, &projectID, &tcJSON); err != nil {
			continue
		}
		var tc map[string]interface{}
		json.Unmarshal(tcJSON, &tc)
		cronExpr, _ := tc["cron"].(string)
		if cronExpr == "" || !w.shouldRun(cronExpr, now) {
			continue
		}
		slog.Info("cron worker: firing workflow", "id", id, "expr", cronExpr)
		if _, err := svc.Execute(ctx, id, projectID, map[string]interface{}{"trigger": "cron"}); err != nil {
			slog.Error("cron worker: fire workflow failed", "workflow_id", id, "error", err)
		}
	}
}

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
		ID: jobID, Type: "deploy_execution",
		Payload: map[string]interface{}{"targetId": targetID, "projectId": projectID, "trigger": "cron"},
	}); err != nil {
		slog.Error("cron worker: push deploy job failed", "target_id", targetID, "error", err)
	}
}

func (w *Cron) fireFunction(ctx context.Context, functionID, projectID string) {
	jobID := functionID + "-cron-" + time.Now().UTC().Format("20060102150405")
	if err := w.queue.Push(ctx, "executions", queue.Job{
		ID: jobID, Type: "function_execution",
		Payload: map[string]interface{}{"functionId": functionID, "projectId": projectID, "trigger": "cron"},
	}); err != nil {
		slog.Error("cron worker: push function job failed", "function_id", functionID, "error", err)
	}
}
