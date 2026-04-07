package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ── In-process fallback (single instance) ────────────────────────────────────

type visitor struct {
	count    int
	lastSeen time.Time
}

type inMemoryLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int
	window   time.Duration
}

func newInMemoryLimiter(rate int, window time.Duration) *inMemoryLimiter {
	rl := &inMemoryLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}
	go func() {
		for {
			time.Sleep(window)
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > window {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *inMemoryLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	v, exists := rl.visitors[ip]
	if !exists || time.Since(v.lastSeen) > rl.window {
		rl.visitors[ip] = &visitor{count: 1, lastSeen: time.Now()}
		return true
	}
	v.count++
	v.lastSeen = time.Now()
	return v.count <= rl.rate
}

// ── Redis-backed limiter (distributed, survives restarts) ────────────────────

// RateLimitRedis returns rate-limit middleware backed by Redis INCR+EXPIRE.
// The window is a fixed 1-minute bucket keyed by IP + unix-minute.
// Falls back gracefully to allowing the request if Redis is unavailable.
func RateLimitRedis(requestsPerMinute int, rdb *redis.Client) func(http.Handler) http.Handler {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 100
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realClientIP(r)
			bucket := time.Now().Unix() / 60
			key := fmt.Sprintf("rl:%s:%d", ip, bucket)

			ctx := r.Context()
			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				// Redis unavailable — fail open (don't block legitimate traffic)
				next.ServeHTTP(w, r)
				return
			}
			if count == 1 {
				rdb.Expire(ctx, key, 2*time.Minute) //nolint:errcheck
			}
			if count > int64(requestsPerMinute) {
				w.Header().Set("Retry-After", "60")
				WriteError(w, http.StatusTooManyRequests,
					"general_rate_limit_exceeded",
					"Rate limit exceeded. Please try again later.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit returns in-process rate-limit middleware.
// Suitable for single-instance deployments only.
// For multi-instance use RateLimitRedis.
func RateLimit(requestsPerMinute int) func(http.Handler) http.Handler {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 100
	}
	rl := newInMemoryLimiter(requestsPerMinute, time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.allow(realClientIP(r)) {
				w.Header().Set("Retry-After", "60")
				WriteError(w, http.StatusTooManyRequests,
					"general_rate_limit_exceeded",
					"Rate limit exceeded. Please try again later.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func realClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.SplitN(fwd, ",", 2)[0]
	}
	if rip := r.Header.Get("X-Real-Ip"); rip != "" {
		return rip
	}
	return r.RemoteAddr
}

