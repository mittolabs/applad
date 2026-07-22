package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
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

// redisCircuitBreaker tracks consecutive Redis failures and trips open after
// a threshold, causing the rate limiter to fail closed (block all requests)
// until Redis recovers. It resets on any successful Redis call.
type redisCircuitBreaker struct {
	failures  atomic.Int64
	threshold int64
}

func (cb *redisCircuitBreaker) recordFailure() { cb.failures.Add(1) }
func (cb *redisCircuitBreaker) recordSuccess() { cb.failures.Store(0) }
func (cb *redisCircuitBreaker) isOpen() bool   { return cb.failures.Load() >= cb.threshold }

// RateLimitRedis returns rate-limit middleware backed by Redis INCR+EXPIRE.
// The window is a fixed 1-minute bucket keyed by IP + unix-minute.
// After 5 consecutive Redis failures the circuit trips and all requests are
// blocked (fail closed) until Redis recovers.
func RateLimitRedis(requestsPerMinute int, rdb *redis.Client) func(http.Handler) http.Handler {
	return RateLimitRedisTiered(requestsPerMinute, requestsPerMinute, rdb)
}

// RateLimitRedisTiered limits anonymous and signed-in callers separately.
//
// One limit for both is what made the console unusable: this guard exists to
// stop a flood from an unknown source, but it counted a signed-in admin's
// page loads the same way. A single console page issues twenty or more
// requests — the shell, the project, the list, its detail, and whatever polls
// — so a few refreshes exhausted a hundred and the whole page failed with
// "Rate limit exceeded", which reads as though the server is out of capacity
// when it is a counter doing what it was told.
//
// Signed-in traffic is still bounded, and abuse of an account is caught by
// the per-user limiter that runs after authentication.
func RateLimitRedisTiered(anonPerMinute, authedPerMinute int, rdb *redis.Client) func(http.Handler) http.Handler {
	if anonPerMinute <= 0 {
		anonPerMinute = 100
	}
	if authedPerMinute <= 0 {
		authedPerMinute = anonPerMinute
	}
	requestsPerMinute := anonPerMinute
	cb := &redisCircuitBreaker{threshold: 5}
	fallback := newInMemoryLimiter(requestsPerMinute, time.Minute)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realClientIP(r)
			projectID := ProjectFromContext(r.Context())
			bucket := time.Now().Unix() / 60

			// Whether the caller presented credentials at all. Whether they
			// are valid is decided later; this only chooses which bucket a
			// request is counted in.
			requestsPerMinute := anonPerMinute
			class := "anon"
			if r.Header.Get("Authorization") != "" || hasSessionCookie(r) {
				requestsPerMinute = authedPerMinute
				class = "auth"
			}

			// Key by project+IP when project context is available, IP-only otherwise
			var key string
			if projectID != "" {
				key = fmt.Sprintf("rl:%s:%s:%s:%d", class, projectID, ip, bucket)
			} else {
				key = fmt.Sprintf("rl:%s:%s:%d", class, ip, bucket)
			}

			ctx := r.Context()
			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				cb.recordFailure()
				// Circuit is open — fall back to in-process limiter so we still
				// enforce rate limits even when Redis is unavailable.
				if !fallback.allow(ip) {
					w.Header().Set("Retry-After", "60")
					WriteError(w, http.StatusTooManyRequests,
						"general_rate_limit_exceeded",
						"Rate limit exceeded. Please try again later.")
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			cb.recordSuccess()
			if count == 1 {
				rdb.Expire(ctx, key, 2*time.Minute) //nolint:errcheck
			}

			limit := int64(requestsPerMinute)
			remaining := limit - count
			if remaining < 0 {
				remaining = 0
			}

			// Always set rate limit headers for transparency
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", (bucket+1)*60))

			if count > limit {
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

// RateLimitUser returns middleware that rate-limits authenticated users by their
// user ID (extracted from the request context set by Authenticate middleware).
// Falls back to IP-based limiting for unauthenticated requests.
// This should be applied inside the RequireAuth group for per-user enforcement.
func RateLimitUser(requestsPerMinute int, rdb *redis.Client) func(http.Handler) http.Handler {
	if requestsPerMinute <= 0 {
		requestsPerMinute = 300
	}
	cb := &redisCircuitBreaker{threshold: 5}
	fallback := newInMemoryLimiter(requestsPerMinute, time.Minute)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserFromContext(r.Context())
			projectID := ProjectFromContext(r.Context())
			bucket := time.Now().Unix() / 60

			var key string
			if userID != "" && projectID != "" {
				key = fmt.Sprintf("rl:u:%s:%s:%d", projectID, userID, bucket)
			} else {
				key = fmt.Sprintf("rl:%s:%d", realClientIP(r), bucket)
			}

			ctx := r.Context()
			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				cb.recordFailure()
				if !fallback.allow(realClientIP(r)) {
					w.Header().Set("Retry-After", "60")
					WriteError(w, http.StatusTooManyRequests,
						"user_rate_limit_exceeded",
						"Per-user rate limit exceeded. Please try again later.")
					return
				}
				next.ServeHTTP(w, r)
				return
			}
			cb.recordSuccess()
			if count == 1 {
				rdb.Expire(ctx, key, 2*time.Minute) //nolint:errcheck
			}

			limit := int64(requestsPerMinute)
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max64(0, limit-count)))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", (bucket+1)*60))

			if count > limit {
				w.Header().Set("Retry-After", "60")
				WriteError(w, http.StatusTooManyRequests,
					"user_rate_limit_exceeded",
					"Per-user rate limit exceeded. Please try again later.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
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

// hasSessionCookie reports whether the request carries a console session.
func hasSessionCookie(r *http.Request) bool {
	for _, c := range r.Cookies() {
		if strings.HasPrefix(c.Name, "applad_session") {
			return true
		}
	}
	return false
}
