package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

/*
 * Limiting what is scarce, rather than counting requests.
 *
 * A request is not a unit of cost. Reading a list is a cached query; starting
 * a deploy is minutes of CPU; sending an SMS is money leaving the account and
 * a sender reputation to lose. One counter over all of them protects the
 * cheapest thing in the system and nothing else — which is exactly what it
 * did: refreshing a page hit the limit while nothing capped builds, messages
 * or password guesses at all.
 *
 * So the expensive operations are named, given their own budget, and keyed by
 * whoever should bear it: a project for work it causes, an account for guesses
 * against it, an address for everything anonymous.
 */

// Scope decides whose budget a request is drawn from.
type Scope string

const (
	// ScopeIP is for callers we cannot identify — the only honest key before
	// authentication.
	ScopeIP Scope = "ip"
	// ScopeProject bills work to the project that caused it, which is also
	// who pays for it.
	ScopeProject Scope = "project"
	// ScopeUser separates people sharing an address, so one busy console does
	// not lock out a colleague behind the same NAT.
	ScopeUser Scope = "user"
	// ScopeAccount keys by the account being acted on rather than the caller,
	// which is what makes a password guess countable: an attacker rotates
	// addresses, but the account under attack stays the same.
	ScopeAccount Scope = "account"
)

// Rule is a budget for one class of operation.
type Rule struct {
	Name      string // namespace for the counter, and what a refusal reports
	Method    string // "" matches any method
	Prefix    string // path prefix, matched after the /v1 mount
	Suffix    string // path suffix, for actions like /trigger
	Scope     Scope
	PerMinute int
	// Message is what the caller is told. A limit they cannot see the reason
	// for is indistinguishable from a broken server.
	Message string
}

func (r Rule) matches(req *http.Request) bool {
	if r.Method != "" && req.Method != r.Method {
		return false
	}
	path := req.URL.Path
	if r.Prefix != "" && !strings.Contains(path, r.Prefix) {
		return false
	}
	if r.Suffix != "" && !strings.HasSuffix(path, r.Suffix) {
		return false
	}
	return true
}

// RateLimitRules enforces per-operation budgets.
//
// Applied where the identity it keys by is already known: after project and
// user context for project work, at the edge for anonymous attempts.
func RateLimitRules(rdb *redis.Client, rules []Rule) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, rule := range rules {
				if !rule.matches(r) {
					continue
				}

				key, ok := ruleKey(r, rule)
				if !ok {
					// No identity for this scope — the generic limiter still
					// applies, and inventing a key would put unrelated callers
					// in one bucket.
					continue
				}

				allowed, retryAfter := consume(r.Context(), rdb, key, rule.PerMinute)
				if !allowed {
					w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
					msg := rule.Message
					if msg == "" {
						msg = "Too many requests. Please try again shortly."
					}
					WriteError(w, http.StatusTooManyRequests, "general_rate_limit_exceeded", msg)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ruleKey builds the counter key, and reports whether this request can be
// attributed to the rule's scope at all.
func ruleKey(r *http.Request, rule Rule) (string, bool) {
	bucket := time.Now().Unix() / 60
	switch rule.Scope {
	case ScopeProject:
		if p := ProjectFromContext(r.Context()); p != "" {
			return fmt.Sprintf("q:%s:p:%s:%d", rule.Name, p, bucket), true
		}
		return "", false
	case ScopeUser:
		if u := UserFromContext(r.Context()); u != "" {
			return fmt.Sprintf("q:%s:u:%s:%d", rule.Name, u, bucket), true
		}
		return "", false
	case ScopeAccount:
		if id := accountIdentifier(r); id != "" {
			return fmt.Sprintf("q:%s:a:%s:%d", rule.Name, strings.ToLower(id), bucket), true
		}
		return "", false
	default:
		return fmt.Sprintf("q:%s:i:%s:%d", rule.Name, realClientIP(r), bucket), true
	}
}

// consume increments a counter and reports whether the request fits.
//
// Redis being unavailable allows the request: this is a fairness mechanism,
// and failing every call closed would turn a cache outage into a total one.
// The generic limiter, which has its own in-process fallback, still applies.
func consume(ctx context.Context, rdb *redis.Client, key string, perMinute int) (bool, int) {
	if rdb == nil || perMinute <= 0 {
		return true, 0
	}
	count, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return true, 0
	}
	if count == 1 {
		rdb.Expire(ctx, key, 2*time.Minute) //nolint:errcheck
	}
	if count > int64(perMinute) {
		return false, 60 - int(time.Now().Unix()%60)
	}
	return true, 0
}

// accountIdentifier reads the account a credential attempt is aimed at.
//
// Read from the body, and put back, because the account under attack is the
// thing worth counting: an attacker rotates addresses freely, so a
// per-address limit on password guessing measures the wrong thing.
func accountIdentifier(r *http.Request) string {
	if r.Body == nil || r.Method != http.MethodPost {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 8<<10))
	if err != nil {
		return ""
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	var payload struct {
		Email  string `json:"email"`
		UserID string `json:"userId"`
		Phone  string `json:"phone"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	switch {
	case payload.Email != "":
		return payload.Email
	case payload.Phone != "":
		return payload.Phone
	default:
		return payload.UserID
	}
}

// AuthRules are the budgets for credential attempts.
//
// Both scopes are needed and neither is sufficient: keyed only by address, an
// attacker rotates addresses; keyed only by account, one address can work
// through a list of accounts. A person who mistypes a password a few times
// is nowhere near these.
func AuthRules() []Rule {
	const tooMany = "Too many attempts. Please wait a minute and try again."
	return []Rule{
		{Name: "auth_ip", Suffix: "/login", Method: http.MethodPost, Scope: ScopeIP, PerMinute: 20, Message: tooMany},
		{Name: "auth_account", Suffix: "/login", Method: http.MethodPost, Scope: ScopeAccount, PerMinute: 10, Message: tooMany},
		{Name: "signup", Suffix: "/signup", Method: http.MethodPost, Scope: ScopeIP, PerMinute: 10, Message: tooMany},
		{Name: "recovery", Prefix: "/recovery", Method: http.MethodPost, Scope: ScopeIP, PerMinute: 10, Message: tooMany},
		{Name: "recovery_account", Prefix: "/recovery", Method: http.MethodPost, Scope: ScopeAccount, PerMinute: 5, Message: tooMany},
		// The console's reset flow lives at /password-reset, not /recovery, so
		// the recovery rules never matched it and it was unlimited.
		{Name: "pwreset", Prefix: "/password-reset", Method: http.MethodPost, Scope: ScopeIP, PerMinute: 5, Message: tooMany},
		{Name: "pwreset_account", Prefix: "/password-reset", Method: http.MethodPost, Scope: ScopeAccount, PerMinute: 5, Message: tooMany},
		{Name: "magic", Prefix: "/magic-url", Method: http.MethodPost, Scope: ScopeIP, PerMinute: 10, Message: tooMany},
		{Name: "verification", Prefix: "/verification", Method: http.MethodPost, Scope: ScopeIP, PerMinute: 10, Message: tooMany},
		{Name: "invite_redeem", Prefix: "/invites/", Method: http.MethodPost, Scope: ScopeIP, PerMinute: 20, Message: tooMany},
	}
}

// ProjectWorkRules are the budgets for work a project causes, as distinct from
// requests it makes.
//
// The numbers are what a person or a reasonable pipeline does, not what the
// hardware survives: a hundred deploys a minute is never intent, and the point
// of stopping it is that each one is minutes of CPU nobody asked for.
func ProjectWorkRules() []Rule {
	return []Rule{
		{Name: "deploy", Suffix: "/trigger", Method: http.MethodPost, Scope: ScopeProject, PerMinute: 10,
			Message: "Too many deploys started in the last minute. Each one is a full build, so they are limited."},
		{Name: "rollback", Suffix: "/rollback", Method: http.MethodPost, Scope: ScopeProject, PerMinute: 10,
			Message: "Too many rollbacks started in the last minute."},
		{Name: "messaging", Prefix: "/messaging/", Method: http.MethodPost, Scope: ScopeProject, PerMinute: 60,
			Message: "Too many messages sent in the last minute. Sending is limited because it costs money and sender reputation."},
		{Name: "functions_exec", Suffix: "/executions", Method: http.MethodPost, Scope: ScopeProject, PerMinute: 300,
			Message: "Too many function executions in the last minute."},
		{Name: "test_runs", Prefix: "/tests/runs", Method: http.MethodPost, Scope: ScopeProject, PerMinute: 20,
			Message: "Too many test runs started in the last minute. Each one builds and runs a container."},
		{Name: "uploads", Prefix: "/storage/", Method: http.MethodPost, Scope: ScopeProject, PerMinute: 120,
			Message: "Too many uploads in the last minute."},
		{Name: "migrations", Prefix: "/migrations", Method: http.MethodPost, Scope: ScopeProject, PerMinute: 10,
			Message: "Too many migration operations in the last minute. Importing a project is expensive, so it is limited."},
	}
}
