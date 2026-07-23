package entitlements

import (
	"context"
	"errors"
	"testing"
)

func reset() { ResetProviders() }

func TestDefaultIsUnlimited(t *testing.T) {
	reset()
	AddProvider(unlimited{})
	d := Get(context.Background(), "o1", "p1")
	if len(d.Limits) != 0 || len(d.Notices) != 0 {
		t.Fatalf("default should withhold nothing and announce nothing, got %+v", d)
	}
	// Non-nil maps so the JSON is {} rather than null.
	if d.Features == nil || d.Limits == nil || d.Notices == nil {
		t.Error("default document must have non-nil collections")
	}
}

type staticProvider struct{ doc Document }

func (s staticProvider) Entitlements(context.Context, string, string) (Document, error) {
	return s.doc, nil
}

func TestProviderSuppliesLimitsAndNotices(t *testing.T) {
	reset()
	AddProvider(staticProvider{doc: Document{
		Limits:  map[string]Limit{"projects": {Limit: 3, Used: 3, Scope: "org"}},
		Notices: []Notice{{ID: "n1", Level: "warn", Title: "Limit reached", Region: RegionAppTop}},
	}})
	defer reset()

	d := Get(context.Background(), "o1", "p1")
	if d.Limits["projects"].Used != 3 {
		t.Errorf("limit not carried: %+v", d.Limits)
	}
	if len(d.Notices) != 1 || d.Notices[0].ID != "n1" {
		t.Errorf("notice not carried: %+v", d.Notices)
	}
}

// A notice naming a region core does not render would otherwise appear
// somewhere unintended, or nowhere, silently.
func TestNoticesWithUnknownRegionAreDropped(t *testing.T) {
	reset()
	AddProvider(staticProvider{doc: Document{
		Notices: []Notice{
			{ID: "good", Level: "info", Title: "ok", Region: RegionPageTop},
			{ID: "bad", Level: "info", Title: "nope", Region: "sidebar.bottom"},
		},
	}})
	defer reset()

	d := Get(context.Background(), "o1", "p1")
	if len(d.Notices) != 1 || d.Notices[0].ID != "good" {
		t.Fatalf("expected only the valid-region notice, got %+v", d.Notices)
	}
}

type flakyProvider struct{ calls int }

func (f *flakyProvider) Entitlements(context.Context, string, string) (Document, error) {
	f.calls++
	if f.calls == 1 {
		return Document{Limits: map[string]Limit{"projects": {Limit: 9, Used: 1, Scope: "org"}}}, nil
	}
	return Document{}, errors.New("metering service unavailable")
}

// Fail open, but prefer last known good so a blip does not hand everyone
// unlimited access.
func TestFailsOpenToLastKnownGood(t *testing.T) {
	reset()
	f := &flakyProvider{}
	AddProvider(f)
	defer reset()

	first := Get(context.Background(), "o1", "p1")
	if first.Limits["projects"].Limit != 9 {
		t.Fatalf("first fetch wrong: %+v", first.Limits)
	}

	Invalidate("") // force a refetch, which now fails
	second := Get(context.Background(), "o1", "p1")
	if second.Limits["projects"].Limit != 9 {
		t.Fatalf("should have served last known good, got %+v", second.Limits)
	}
}

func TestFailsOpenToUnlimitedWithoutCache(t *testing.T) {
	reset()
	AddProvider(&flakyProvider{calls: 1}) // every call errors
	defer reset()

	d := Get(context.Background(), "o1", "p1")
	if len(d.Limits) != 0 {
		t.Fatalf("expected unlimited fallback, got %+v", d.Limits)
	}
}

// An upgrade must take effect at once, not after the TTL.
func TestInvalidateForcesRefetch(t *testing.T) {
	reset()
	p := staticProvider{doc: Document{Limits: map[string]Limit{"projects": {Limit: 1, Scope: "org"}}}}
	AddProvider(p)
	_ = Get(context.Background(), "o1", "p1")

	AddProvider(staticProvider{doc: Document{Limits: map[string]Limit{"projects": {Limit: 50, Scope: "org"}}}})
	defer reset()

	d := Get(context.Background(), "o1", "p1")
	if d.Limits["projects"].Limit != 50 {
		t.Fatalf("SetProvider should invalidate; got %+v", d.Limits)
	}
}

// Two modules legitimately contribute to one subject: billing knows the plan's
// limits, another knows what to announce. Before this, the last registration
// silently won and the other module's answer vanished.
func TestContributorsMerge(t *testing.T) {
	reset()
	defer reset()
	AddProvider(staticProvider{doc: Document{
		Limits: map[string]Limit{"projects": {Limit: 3, Used: 1, Scope: "org"}},
	}})
	AddProvider(staticProvider{doc: Document{
		Notices: []Notice{{ID: "n1", Level: "info", Title: "hello", Region: RegionAppTop}},
	}})

	d := Get(context.Background(), "o1", "p1")
	if d.Limits["projects"].Limit != 3 {
		t.Errorf("limit from the first contributor lost: %+v", d.Limits)
	}
	if len(d.Notices) != 1 {
		t.Errorf("notice from the second contributor lost: %+v", d.Notices)
	}
}

// One contributor falling over must not withhold what the others know.
func TestOneFailingContributorDoesNotSilenceTheRest(t *testing.T) {
	reset()
	defer reset()
	AddProvider(&flakyProvider{calls: 1}) // always errors
	AddProvider(staticProvider{doc: Document{
		Limits: map[string]Limit{"projects": {Limit: 9, Scope: "org"}},
	}})

	d := Get(context.Background(), "o1", "p1")
	if d.Limits["projects"].Limit != 9 {
		t.Fatalf("a healthy contributor was silenced by a failing one: %+v", d.Limits)
	}
}
