// Package appcache provides a managed key-value cache API backed by Redis.
// It lives under /v1/cache and is separate from the internal Redis cache package.
package appcache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Entry is a single cache entry returned to callers.
type Entry struct {
	Key       string        `json:"key"`
	Value     interface{}   `json:"value"`
	Tags      []string      `json:"tags"`
	TTL       time.Duration `json:"ttl"`
	ExpiresAt *time.Time    `json:"expiresAt,omitempty"`
}

// Service wraps Redis to expose a project-scoped managed cache.
type Service struct {
	rdb *redis.Client
}

// NewService creates a new cache Service.
func NewService(rdb *redis.Client) *Service {
	return &Service{rdb: rdb}
}

// Set stores a value at key with optional TTL (0 = no expiry) and tags.
func (s *Service) Set(ctx context.Context, projectID, key string, value interface{}, ttl time.Duration, tags []string) error {
	rkey := s.rkey(projectID, key)
	payload, err := json.Marshal(map[string]interface{}{"v": value, "tags": tags})
	if err != nil {
		return fmt.Errorf("cache: marshal: %w", err)
	}
	if err := s.rdb.Set(ctx, rkey, payload, ttl).Err(); err != nil {
		return fmt.Errorf("cache: set: %w", err)
	}
	// Index each tag → set of keys (for tag-based invalidation)
	for _, tag := range tags {
		s.rdb.SAdd(ctx, s.tagKey(projectID, tag), rkey) //nolint:errcheck
		if ttl > 0 {
			s.rdb.Expire(ctx, s.tagKey(projectID, tag), ttl+time.Minute) //nolint:errcheck
		}
	}
	return nil
}

// Get retrieves a cache entry by key.
func (s *Service) Get(ctx context.Context, projectID, key string) (*Entry, error) {
	rkey := s.rkey(projectID, key)
	raw, err := s.rdb.Get(ctx, rkey).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cache: get: %w", err)
	}
	var payload struct {
		V    interface{} `json:"v"`
		Tags []string    `json:"tags"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("cache: unmarshal: %w", err)
	}
	entry := &Entry{Key: key, Value: payload.V, Tags: payload.Tags}
	d, err := s.rdb.TTL(ctx, rkey).Result()
	if err == nil && d > 0 {
		entry.TTL = d
		t := time.Now().Add(d)
		entry.ExpiresAt = &t
	}
	return entry, nil
}

// Delete removes a single cache entry.
func (s *Service) Delete(ctx context.Context, projectID, key string) error {
	return s.rdb.Del(ctx, s.rkey(projectID, key)).Err()
}

// InvalidateByTag deletes all entries tagged with the given tag.
func (s *Service) InvalidateByTag(ctx context.Context, projectID, tag string) (int, error) {
	tagKey := s.tagKey(projectID, tag)
	members, err := s.rdb.SMembers(ctx, tagKey).Result()
	if err != nil {
		return 0, err
	}
	if len(members) == 0 {
		return 0, nil
	}
	keys := make([]string, len(members))
	copy(keys, members)
	keys = append(keys, tagKey)
	count, err := s.rdb.Del(ctx, keys...).Result()
	return int(count) - 1, err
}

// Flush deletes all cache entries for a project.
func (s *Service) Flush(ctx context.Context, projectID string) (int, error) {
	pattern := fmt.Sprintf("cache:%s:*", projectID)
	var keys []string
	iter := s.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return 0, err
	}
	if len(keys) == 0 {
		return 0, nil
	}
	count, err := s.rdb.Del(ctx, keys...).Result()
	return int(count), err
}

// Stats returns hit/miss stats for a project (best-effort; requires keyspace notifications).
func (s *Service) Stats(ctx context.Context, projectID string) (map[string]interface{}, error) {
	pattern := fmt.Sprintf("cache:%s:*", projectID)
	var count int64
	iter := s.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if !strings.Contains(iter.Val(), ":tag:") {
			count++
		}
	}
	return map[string]interface{}{"entries": count}, iter.Err()
}

func (s *Service) rkey(projectID, key string) string {
	return fmt.Sprintf("cache:%s:%s", projectID, key)
}

func (s *Service) tagKey(projectID, tag string) string {
	return fmt.Sprintf("cache:%s:tag:%s", projectID, tag)
}
