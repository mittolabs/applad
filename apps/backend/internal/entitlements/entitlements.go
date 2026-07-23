// Package entitlements is the document the console reads to know what it may
// offer and what to tell the user.
//
// The vocabulary is deliberately generic: features, limits, notices. Core never
// learns the words plan, subscription or Stripe. The default provider returns
// unlimited with no notices, which is not a stub but the right answer for an
// install nobody is metering.
package entitlements

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Limit is a metered allowance and its current consumption.
type Limit struct {
	Limit int64  `json:"limit"`
	Used  int64  `json:"used"`
	Scope string `json:"scope"` // org | project
}

// Action is the next step offered by a notice.
type Action struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

// Notice is a banner. It is DATA, rendered by core in core styling at a named
// region. Providers never inject markup: that is what keeps a composed build
// from drifting away from the core design, and keeps core refactors from
// breaking whoever supplies the notice.
type Notice struct {
	ID          string  `json:"id"`
	Level       string  `json:"level"` // info | warn | critical
	Title       string  `json:"title"`
	Body        string  `json:"body,omitempty"`
	Action      *Action `json:"action,omitempty"`
	Dismissible bool    `json:"dismissible"`
	Scope       string  `json:"scope,omitempty"` // org | project
	Region      string  `json:"region"`          // app.top | page.top | project.top
	// Theme is optional presentation, chosen from a vocabulary core validates
	// and renders. Never markup: see theme.go for why.
	Theme *Theme `json:"theme,omitempty"`
}

// Valid notice regions. A notice naming anything else is dropped rather than
// rendered somewhere unintended.
const (
	RegionAppTop     = "app.top"
	RegionPageTop    = "page.top"
	RegionProjectTop = "project.top"
)

func validRegion(r string) bool {
	return r == RegionAppTop || r == RegionPageTop || r == RegionProjectTop
}

// Document is the full entitlements payload.
type Document struct {
	Features map[string]bool  `json:"features"`
	Limits   map[string]Limit `json:"limits"`
	Notices  []Notice         `json:"notices"`
}

// Unlimited is the default document: nothing withheld, nothing to announce.
func Unlimited() Document {
	return Document{
		Features: map[string]bool{},
		Limits:   map[string]Limit{},
		Notices:  []Notice{},
	}
}

// Provider produces entitlements for a subject.
type Provider interface {
	Entitlements(ctx context.Context, orgID, projectID string) (Document, error)
}

type unlimited struct{}

func (unlimited) Entitlements(context.Context, string, string) (Document, error) {
	return Unlimited(), nil
}

var (
	mu sync.RWMutex
	// Several modules legitimately have something to say about one subject:
	// billing knows the plan's limits, another module knows what to announce.
	// A single provider slot made the last registration silently win.
	providers []Provider

	cacheMu  sync.RWMutex
	cache    = map[string]cached{}
	cacheTTL = 60 * time.Second
)

type cached struct {
	doc Document
	at  time.Time
}

// AddProvider registers a contributor. Registered at startup, before serving.
// Cached documents are discarded: a new contributor changes the answer.
func AddProvider(p Provider) {
	if p == nil {
		return
	}
	mu.Lock()
	providers = append(providers, p)
	mu.Unlock()

	cacheMu.Lock()
	cache = map[string]cached{}
	cacheMu.Unlock()
}

// ResetProviders drops every contributor. For tests.
func ResetProviders() {
	mu.Lock()
	providers = nil
	mu.Unlock()
	cacheMu.Lock()
	cache = map[string]cached{}
	cacheMu.Unlock()
}

// merge folds one contributor's answer into the document being built. Limits and
// features are keyed, so a later contributor overrides the same key; notices
// accumulate, because two modules announcing different things both have a point.
func merge(into *Document, from Document) {
	for k, v := range from.Features {
		into.Features[k] = v
	}
	for k, v := range from.Limits {
		into.Limits[k] = v
	}
	into.Notices = append(into.Notices, from.Notices...)
}

// Invalidate marks cached entitlements stale so the next read refetches. An
// empty key marks everything.
//
// It deliberately RETAINS the last document rather than deleting it. Deleting
// would mean a provider that fails immediately after an upgrade drops the
// subject all the way to unlimited, which is the expensive direction to be
// wrong in. Stale-but-real beats nothing.
func Invalidate(key string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if key == "" {
		for k, c := range cache {
			c.at = time.Time{}
			cache[k] = c
		}
		return
	}
	if c, ok := cache[key]; ok {
		c.at = time.Time{}
		cache[key] = c
	}
}

// Get returns entitlements for a subject.
//
// It fails OPEN: a provider that errors serves the last known good document if
// one is cached, and unlimited otherwise. Withholding features because a
// metering service is unwell would turn its outage into everyone's.
func Get(ctx context.Context, orgID, projectID string) Document {
	key := orgID + "/" + projectID

	cacheMu.RLock()
	c, ok := cache[key]
	cacheMu.RUnlock()
	if ok && time.Since(c.at) < cacheTTL {
		return c.doc
	}

	mu.RLock()
	ps := make([]Provider, len(providers))
	copy(ps, providers)
	mu.RUnlock()

	if len(ps) == 0 {
		return Unlimited()
	}

	doc := Unlimited()
	failed := 0
	for _, p := range ps {
		part, err := p.Entitlements(ctx, orgID, projectID)
		if err != nil {
			// One contributor failing must not withhold what the others know.
			slog.Error("entitlements: contributor failed", "error", err)
			failed++
			continue
		}
		merge(&doc, part)
	}
	if failed == len(ps) {
		slog.Error("entitlements: every contributor failed, serving last known good")
		if ok {
			return c.doc
		}
		return Unlimited()
	}
	doc = normalise(doc)

	cacheMu.Lock()
	cache[key] = cached{doc: doc, at: time.Now()}
	cacheMu.Unlock()
	return doc
}

// normalise guarantees non-nil maps for the JSON payload and drops notices that
// name an unknown region.
func normalise(d Document) Document {
	if d.Features == nil {
		d.Features = map[string]bool{}
	}
	if d.Limits == nil {
		d.Limits = map[string]Limit{}
	}
	kept := make([]Notice, 0, len(d.Notices))
	for _, n := range d.Notices {
		if !validRegion(n.Region) {
			slog.Warn("entitlements: dropping notice with unknown region", "id", n.ID, "region", n.Region)
			continue
		}
		n.Theme = n.Theme.sanitise()
		kept = append(kept, n)
	}
	d.Notices = kept
	return d
}
