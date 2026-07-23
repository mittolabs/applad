// Package policy is the quota and permission seam.
//
// Core declares WHERE a gate applies by naming a capability at each action that
// creates or consumes a resource. Something else decides WHETHER it passes: the
// default resolver allows everything, which is the correct answer for an install
// nobody is metering. An operator who wants internal quotas can register their
// own resolver.
//
// Enforcement here is authoritative. A disabled button in a UI is a hint; this
// is the boundary.
package policy

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// ScopeKind is what a capability is counted against.
type ScopeKind string

const (
	ScopeOrg     ScopeKind = "org"
	ScopeProject ScopeKind = "project"
)

// Scope identifies the subject a decision is made for.
type Scope struct {
	Kind ScopeKind `json:"kind"`
	ID   string    `json:"id"`
}

// Org scopes a decision to an organization.
func Org(id string) Scope { return Scope{Kind: ScopeOrg, ID: id} }

// Project scopes a decision to a project.
func Project(id string) Scope { return Scope{Kind: ScopeProject, ID: id} }

// Action is what the caller can do about a denial, supplied by the resolver.
type Action struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

// Decision is the answer for one capability check.
type Decision struct {
	Allowed bool    `json:"allowed"`
	Reason  string  `json:"reason,omitempty"`
	Action  *Action `json:"action,omitempty"`
}

// Allow is the permissive decision.
var Allow = Decision{Allowed: true}

// Deny builds a denial carrying a human reason and an optional next step.
func Deny(reason string, action *Action) Decision {
	return Decision{Allowed: false, Reason: reason, Action: action}
}

// Resolver decides capability checks.
type Resolver interface {
	Decide(ctx context.Context, key string, scope Scope) Decision
}

// allowAll is the default: no metering, everything permitted.
type allowAll struct{}

func (allowAll) Decide(context.Context, string, Scope) Decision { return Allow }

var (
	mu       sync.RWMutex
	resolver Resolver = allowAll{}
)

// SetResolver installs the resolver. Registered at startup, before serving.
func SetResolver(r Resolver) {
	mu.Lock()
	defer mu.Unlock()
	if r == nil {
		r = allowAll{}
	}
	resolver = r
}

// Decide evaluates a capability.
//
// It fails OPEN: an unknown key, or a resolver that panics, allows the action
// and logs loudly. A quota layer is about fairness and revenue, not
// authorisation, so a broken resolver must not become a total outage. This
// matches how the rate limiter treats an unreachable Redis.
func Decide(ctx context.Context, key string, scope Scope) (d Decision) {
	if !Known(key) {
		slog.Error("policy: unknown capability, allowing", "key", key)
		return Allow
	}
	mu.RLock()
	r := resolver
	mu.RUnlock()

	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("policy: resolver panicked, allowing", "key", key, "panic", rec)
			d = Allow
		}
	}()
	return r.Decide(ctx, key, scope)
}

// DeniedError is returned by Require when a capability is refused. Handlers
// translate it into 402 with the reason and action intact.
type DeniedError struct {
	Key      string
	Decision Decision
}

func (e *DeniedError) Error() string {
	if e.Decision.Reason != "" {
		return e.Decision.Reason
	}
	return fmt.Sprintf("%s is not permitted", e.Key)
}

// Require evaluates a capability and returns *DeniedError when refused.
func Require(ctx context.Context, key string, scope Scope) error {
	if d := Decide(ctx, key, scope); !d.Allowed {
		return &DeniedError{Key: key, Decision: d}
	}
	return nil
}
