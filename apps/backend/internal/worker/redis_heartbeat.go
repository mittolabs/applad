package worker

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// StartRedisHeartbeat publishes a liveness key (status:worker:<name>) to Redis
// every 30s with a 90s TTL, so the API's self-monitoring can tell which workers
// are alive (the key disappears shortly after a worker dies). Call once from a
// worker's Start after its Redis client is ready.
func StartRedisHeartbeat(ctx context.Context, rdb *redis.Client, name string) {
	key := "status:worker:" + name
	beat := func() {
		rdb.Set(ctx, key, time.Now().UTC().Format(time.RFC3339), 90*time.Second)
	}
	beat()
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				beat()
			}
		}
	}()
}
