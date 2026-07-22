package oauth

import (
	"strings"
	"testing"
)

func TestGetAuthURL_ContainsClientID(t *testing.T) {
	p := &Provider{
		Name:     "test",
		ClientID: "my-client-id-123",
		AuthURL:  "https://example.com/auth",
		Scopes:   []string{"openid"},
	}

	u := p.GetAuthURL("https://example.com/callback", "random-state")

	if !strings.Contains(u, "my-client-id-123") {
		t.Errorf("auth URL should contain client_id, got: %s", u)
	}
	if !strings.Contains(u, "example.com%2Fcallback") && !strings.Contains(u, "example.com/callback") {
		t.Errorf("auth URL should contain redirect_uri, got: %s", u)
	}
}

func TestGetAuthURL_ContainsScopes(t *testing.T) {
	p := &Provider{
		Name:     "test",
		ClientID: "cid",
		AuthURL:  "https://example.com/auth",
		Scopes:   []string{"openid", "email", "profile"},
	}

	u := p.GetAuthURL("https://example.com/cb", "state1")

	// Scopes are joined with space, which gets URL-encoded as +
	if !strings.Contains(u, "openid") {
		t.Errorf("auth URL should contain 'openid' scope, got: %s", u)
	}
	if !strings.Contains(u, "email") {
		t.Errorf("auth URL should contain 'email' scope, got: %s", u)
	}
	if !strings.Contains(u, "profile") {
		t.Errorf("auth URL should contain 'profile' scope, got: %s", u)
	}
}

func TestParseGoogleUser(t *testing.T) {
	body := []byte(`{"id":"123","email":"a@b.com","name":"Alice"}`)
	u, err := parseGoogleUser(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != "123" {
		t.Errorf("expected ID '123', got '%s'", u.ID)
	}
	if u.Email != "a@b.com" {
		t.Errorf("expected email 'a@b.com', got '%s'", u.Email)
	}
	if u.Name != "Alice" {
		t.Errorf("expected name 'Alice', got '%s'", u.Name)
	}
	if u.Provider != "google" {
		t.Errorf("expected provider 'google', got '%s'", u.Provider)
	}
}

func TestParseGitHubUser(t *testing.T) {
	body := []byte(`{"id":456,"email":"b@c.com","name":"Bob","login":"bob"}`)
	u, err := parseGitHubUser(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != "456" {
		t.Errorf("expected ID '456', got '%s'", u.ID)
	}
	if u.Name != "Bob" {
		t.Errorf("expected name 'Bob', got '%s'", u.Name)
	}
	if u.Provider != "github" {
		t.Errorf("expected provider 'github', got '%s'", u.Provider)
	}
}

func TestParseGitHubUser_FallbackToLogin(t *testing.T) {
	body := []byte(`{"id":789,"email":"c@d.com","name":"","login":"charlie"}`)
	u, err := parseGitHubUser(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Name != "charlie" {
		t.Errorf("expected name to fall back to login 'charlie', got '%s'", u.Name)
	}
}

func TestParseDiscordUser(t *testing.T) {
	body := []byte(`{"id":"111","email":"d@e.com","username":"disco","global_name":"Disco Dan"}`)
	u, err := parseDiscordUser(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != "111" {
		t.Errorf("expected ID '111', got '%s'", u.ID)
	}
	if u.Email != "d@e.com" {
		t.Errorf("expected email 'd@e.com', got '%s'", u.Email)
	}
	if u.Name != "Disco Dan" {
		t.Errorf("expected name 'Disco Dan', got '%s'", u.Name)
	}
	if u.Provider != "discord" {
		t.Errorf("expected provider 'discord', got '%s'", u.Provider)
	}
}

func TestParseDiscordUser_FallbackToUsername(t *testing.T) {
	body := []byte(`{"id":"222","email":"e@f.com","username":"fallback_user","global_name":""}`)
	u, err := parseDiscordUser(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Name != "fallback_user" {
		t.Errorf("expected name to fall back to username 'fallback_user', got '%s'", u.Name)
	}
}

func TestParseGenericUser(t *testing.T) {
	parser := parseGenericUser("testprovider")
	body := []byte(`{"id":"gen1","email":"g@h.com","name":"Generic"}`)
	u, err := parser(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != "gen1" {
		t.Errorf("expected ID 'gen1', got '%s'", u.ID)
	}
	if u.Email != "g@h.com" {
		t.Errorf("expected email 'g@h.com', got '%s'", u.Email)
	}
	if u.Name != "Generic" {
		t.Errorf("expected name 'Generic', got '%s'", u.Name)
	}
	if u.Provider != "testprovider" {
		t.Errorf("expected provider 'testprovider', got '%s'", u.Provider)
	}
}

func TestParseGenericUser_SubFallback(t *testing.T) {
	parser := parseGenericUser("subprovider")
	body := []byte(`{"sub":"sub-id-99","email":"s@t.com","name":"SubUser"}`)
	u, err := parser(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != "sub-id-99" {
		t.Errorf("expected ID 'sub-id-99' from sub field, got '%s'", u.ID)
	}
}

func TestAllProviderDefinitions_HasExpectedProviders(t *testing.T) {
	defs := AllProviderDefinitions()

	expected := []string{"google", "github", "apple", "facebook", "discord", "twitter", "microsoft"}
	for _, name := range expected {
		if _, ok := defs[name]; !ok {
			t.Errorf("expected provider '%s' in AllProviderDefinitions, but not found", name)
		}
	}
}

func TestListAvailableProviders(t *testing.T) {
	providers := ListAvailableProviders()
	if len(providers) == 0 {
		t.Error("expected non-empty list of available providers")
	}

	// Should have at least the 7 core providers
	if len(providers) < 7 {
		t.Errorf("expected at least 7 providers, got %d", len(providers))
	}
}

func TestToProvider_CreatesConfiguredProvider(t *testing.T) {
	defs := AllProviderDefinitions()
	def := defs["google"]
	p := def.ToProvider("my-id", "my-secret")

	if p.ClientID != "my-id" {
		t.Errorf("expected ClientID 'my-id', got '%s'", p.ClientID)
	}
	if p.ClientSecret != "my-secret" {
		t.Errorf("expected ClientSecret 'my-secret', got '%s'", p.ClientSecret)
	}
	if p.Name != "Google" {
		t.Errorf("expected Name 'Google', got '%s'", p.Name)
	}
}
