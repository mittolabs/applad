package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache wraps the Redis client.
type Cache struct {
	client *redis.Client
}

// Connect creates a Redis client and verifies connectivity.
func Connect(addr string) (*Cache, error) {
	c := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("cache: ping: %w", err)
	}
	return &Cache{client: c}, nil
}

// Client returns the underlying Redis client.
func (c *Cache) Client() *redis.Client { return c.client }

// Ping checks that Redis is reachable.
func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
