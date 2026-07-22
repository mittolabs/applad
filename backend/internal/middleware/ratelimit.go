package middleware

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
	// No secret: both tiers are the same size, so nothing needs validating.
	return RateLimitRedisTiered(requestsPerMinute, requestsPerMinute, rdb, "")
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
//
// jwtSecret is what makes the tiers honest: the larger bucket goes only to a
// caller whose token actually verifies, not to anyone who sends a junk
// Authorization header.
func RateLimitRedisTiered(anonPerMinute, authedPerMinute int, rdb *redis.Client, jwtSecret string) func(http.Handler) http.Handler {
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

			// The authed bucket requires a token that verifies. Presence of a
			// header used to be enough, which let anyone claim the larger
			// budget for free.
			requestsPerMinute := anonPerMinute
			class := "anon"
			if validSessionToken(r, jwtSecret) {
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

// trustedProxyNets is the set of peers whose forwarding headers are believed.
// Anyone can write X-Forwarded-For; believing it from everyone let a single
// host rotate identities past every IP limit. Defaults cover our own proxy:
// the docker bridge range and loopback. Override with TRUSTED_PROXY_CIDRS.
var trustedProxyNets = parseTrustedProxies(os.Getenv("TRUSTED_PROXY_CIDRS"))

func parseTrustedProxies(env string) []*net.IPNet {
	if strings.TrimSpace(env) == "" {
		env = "172.16.0.0/12,127.0.0.1/32,::1/128"
	}
	var nets []*net.IPNet
	for _, part := range strings.Split(env, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// A bare address is shorthand for a single-host network.
		if !strings.Contains(part, "/") {
			if ip := net.ParseIP(part); ip != nil {
				bits := 32
				if ip.To4() == nil {
					bits = 128
				}
				part = fmt.Sprintf("%s/%d", part, bits)
			}
		}
		if _, n, err := net.ParseCIDR(part); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}

// trustedPeer reports whether the direct TCP peer is one of our proxies.
func trustedPeer(remoteAddr string) bool {
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, n := range trustedProxyNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// forwardedIP returns the address the trusted proxy saw. The LAST entry of
// X-Forwarded-For is the one our proxy appended; anything before it arrived
// from the client and is attacker-controlled.
func forwardedIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return strings.TrimSpace(r.Header.Get("X-Real-Ip"))
}

// RealIP rewrites RemoteAddr from forwarding headers, but only when the peer
// is a trusted proxy. Replaces chi's RealIP, which believed anyone.
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if trustedPeer(r.RemoteAddr) {
			if ip := forwardedIP(r); ip != "" {
				r.RemoteAddr = ip
			}
		}
		next.ServeHTTP(w, r)
	})
}

// realClientIP resolves who to bill a request to: the forwarded address when
// the peer is a proxy we trust, the peer itself otherwise. The port is
// stripped so one host is one bucket, not one bucket per connection.
func realClientIP(r *http.Request) string {
	if trustedPeer(r.RemoteAddr) {
		if ip := forwardedIP(r); ip != "" {
			return ip
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// ClientIP is realClientIP for packages that record addresses (session logs).
func ClientIP(r *http.Request) string { return realClientIP(r) }

// hasSessionCookie reports whether the request carries anything session-shaped.
// Real session cookies are "a_session*"; "applad_session" is the JS-readable
// signed-in hint. Presence proves nothing — the authed tier requires the
// token to verify (validSessionToken).
func hasSessionCookie(r *http.Request) bool {
	for _, c := range r.Cookies() {
		if strings.HasPrefix(c.Name, "a_session") || c.Name == "applad_session" {
			return true
		}
	}
	return false
}

// validSessionToken reports whether any presented credential is a JWT we
// signed. HS256 only, matching how every session token is issued.
func validSessionToken(r *http.Request, secret string) bool {
	if secret == "" {
		return false
	}
	var candidates []string
	if auth := r.Header.Get("Authorization"); auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			candidates = append(candidates, parts[1])
		}
	}
	for _, c := range r.Cookies() {
		if strings.HasPrefix(c.Name, "a_session") && c.Value != "" {
			candidates = append(candidates, c.Value)
		}
	}
	for _, tok := range candidates {
		t, err := jwt.Parse(tok, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{"HS256"}))
		if err == nil && t.Valid {
			return true
		}
	}
	return false
}
