package appcache

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// integrationClient returns a Redis client if REDIS_ADDR is set; skips otherwise.
func skipIfNoRedis(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Redis unavailable (%v) — skipping integration test", err)
	}
	return rdb
}

func TestSetGet(t *testing.T) {
	rdb := skipIfNoRedis(t)
	svc := NewService(rdb)
	ctx := context.Background()
	proj := "test-cache-proj"

	if err := svc.Set(ctx, proj, "hello", "world", 5*time.Second, nil); err != nil {
		t.Fatalf("Set: %v", err)
	}
	entry, err := svc.Get(ctx, proj, "hello")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if entry.Value.(string) != "world" {
		t.Errorf("value = %v, want world", entry.Value)
	}
}

func TestGetMiss(t *testing.T) {
	rdb := skipIfNoRedis(t)
	svc := NewService(rdb)

	entry, err := svc.Get(context.Background(), "proj", "nonexistent-key-xyz")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if entry != nil {
		t.Error("expected nil for cache miss")
	}
}

func TestDelete(t *testing.T) {
	rdb := skipIfNoRedis(t)
	svc := NewService(rdb)
	ctx := context.Background()
	proj := "test-cache-del"

	svc.Set(ctx, proj, "k", "v", time.Minute, nil) //nolint:errcheck
	svc.Delete(ctx, proj, "k")                       //nolint:errcheck

	entry, _ := svc.Get(ctx, proj, "k")
	if entry != nil {
		t.Error("expected nil after delete")
	}
}

func TestInvalidateByTag(t *testing.T) {
	rdb := skipIfNoRedis(t)
	svc := NewService(rdb)
	ctx := context.Background()
	proj := "test-cache-tag"

	svc.Set(ctx, proj, "a", 1, time.Minute, []string{"tag1"}) //nolint:errcheck
	svc.Set(ctx, proj, "b", 2, time.Minute, []string{"tag1"}) //nolint:errcheck
	svc.Set(ctx, proj, "c", 3, time.Minute, []string{"tag2"}) //nolint:errcheck

	n, err := svc.InvalidateByTag(ctx, proj, "tag1")
	if err != nil {
		t.Fatalf("InvalidateByTag: %v", err)
	}
	if n != 2 {
		t.Errorf("invalidated %d, want 2", n)
	}
	// tag2 key should still exist
	entry, _ := svc.Get(ctx, proj, "c")
	if entry == nil {
		t.Error("tag2 key should survive tag1 invalidation")
	}
}

func TestRkeyNamespacing(t *testing.T) {
	svc := &Service{rdb: nil}
	k := svc.rkey("projA", "key1")
	if k != "cache:projA:key1" {
		t.Errorf("unexpected rkey: %s", k)
	}
	tk := svc.tagKey("projA", "tag1")
	if tk != "cache:projA:tag:tag1" {
		t.Errorf("unexpected tagKey: %s", tk)
	}
}
