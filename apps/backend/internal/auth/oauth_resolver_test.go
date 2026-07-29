package auth

import (
	"context"
	"testing"
)

type stubProvider struct{ id string }

func (p stubProvider) GetAuthURL(redirectURI, state string) string { return "https://auth/" + p.id }
func (p stubProvider) ExchangeCode(_ context.Context, _, _ string) (string, error) {
	return "tok-" + p.id, nil
}
func (p stubProvider) GetUserInfo(_ context.Context, _ string) (OAuthUserInfo, error) {
	return OAuthUserInfo{ID: p.id}, nil
}

type stubResolver struct{ byProject map[string]OAuthProvider }

func (r stubResolver) ResolveOAuthProvider(_ context.Context, projectID, _ string) OAuthProvider {
	return r.byProject[projectID]
}

// A per-project resolver, when it returns a provider, wins over the static map.
func TestResolveOAuthProvider_ResolverWins(t *testing.T) {
	h := &Handler{}
	h.SetOAuthProviders(map[string]OAuthProvider{"google": stubProvider{id: "global"}})
	h.SetOAuthResolver(stubResolver{byProject: map[string]OAuthProvider{
		"proj1": stubProvider{id: "project"},
	}})

	p, ok := h.resolveOAuthProvider(context.Background(), "proj1", "google")
	if !ok {
		t.Fatal("expected a provider")
	}
	if sp, _ := p.(stubProvider); sp.id != "project" {
		t.Fatalf("expected per-project provider, got %+v", p)
	}
}

// When the resolver has nothing for the project, it falls back to the map.
func TestResolveOAuthProvider_FallsBackToMap(t *testing.T) {
	h := &Handler{}
	h.SetOAuthProviders(map[string]OAuthProvider{"google": stubProvider{id: "global"}})
	h.SetOAuthResolver(stubResolver{byProject: map[string]OAuthProvider{}})

	p, ok := h.resolveOAuthProvider(context.Background(), "projX", "google")
	if !ok {
		t.Fatal("expected fallback provider")
	}
	if sp, _ := p.(stubProvider); sp.id != "global" {
		t.Fatalf("expected global provider, got %+v", p)
	}
}

// With no resolver installed at all (e.g. a bare handler), the map still works.
func TestResolveOAuthProvider_NoResolver(t *testing.T) {
	h := &Handler{}
	h.SetOAuthProviders(map[string]OAuthProvider{"github": stubProvider{id: "gh"}})

	if _, ok := h.resolveOAuthProvider(context.Background(), "proj1", "github"); !ok {
		t.Fatal("expected provider from map")
	}
	if _, ok := h.resolveOAuthProvider(context.Background(), "proj1", "nope"); ok {
		t.Fatal("unknown provider should not resolve")
	}
}
