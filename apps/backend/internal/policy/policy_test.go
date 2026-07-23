package policy

import (
	"context"
	"errors"
	"testing"
)

func reset() { SetResolver(nil) }

func TestDefaultAllowsEverything(t *testing.T) {
	reset()
	for _, c := range Capabilities() {
		if err := Require(context.Background(), c.Key, Org("o1")); err != nil {
			t.Fatalf("default resolver denied %s: %v", c.Key, err)
		}
	}
}

type denyAll struct{}

func (denyAll) Decide(context.Context, string, Scope) Decision {
	return Deny("Project limit reached (3/3)", &Action{Label: "Upgrade", Href: "/billing"})
}

func TestResolverCanDeny(t *testing.T) {
	reset()
	SetResolver(denyAll{})
	defer reset()

	err := Require(context.Background(), "projects.create", Org("o1"))
	if err == nil {
		t.Fatal("expected denial")
	}
	var denied *DeniedError
	if !errors.As(err, &denied) {
		t.Fatalf("expected *DeniedError, got %T", err)
	}
	if denied.Decision.Reason != "Project limit reached (3/3)" {
		t.Errorf("reason not carried: %q", denied.Decision.Reason)
	}
	if denied.Decision.Action == nil || denied.Decision.Action.Href != "/billing" {
		t.Error("action not carried to the caller")
	}
}

type panicker struct{}

func (panicker) Decide(context.Context, string, Scope) Decision { panic("resolver exploded") }

// A quota layer is about fairness and revenue, not authorisation. A resolver
// that falls over must not take the product down with it.
func TestFailsOpenWhenResolverPanics(t *testing.T) {
	reset()
	SetResolver(panicker{})
	defer reset()

	if err := Require(context.Background(), "projects.create", Org("o1")); err != nil {
		t.Fatalf("expected fail-open, got denial: %v", err)
	}
}

func TestUnknownCapabilityFailsOpen(t *testing.T) {
	reset()
	SetResolver(denyAll{})
	defer reset()

	// Not in the registry: allowed rather than silently blocking a core action.
	if err := Require(context.Background(), "totally.unknown", Org("o1")); err != nil {
		t.Fatalf("unknown key should fail open, got %v", err)
	}
}

// The registry is a contract. Keys must be unique, scoped, and described.
func TestRegistryIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Capabilities() {
		if seen[c.Key] {
			t.Errorf("duplicate capability key %q", c.Key)
		}
		seen[c.Key] = true
		if c.Scope != ScopeOrg && c.Scope != ScopeProject {
			t.Errorf("%s has invalid scope %q", c.Key, c.Scope)
		}
		if c.Desc == "" {
			t.Errorf("%s has no description", c.Key)
		}
		if !Known(c.Key) {
			t.Errorf("%s not resolvable via Known", c.Key)
		}
	}
}
