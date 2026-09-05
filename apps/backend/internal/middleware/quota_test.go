package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The limiter that existed counted every call the same, so refreshing a page
// hit it while nothing capped a deploy. These are the rules that decide which
// operations are expensive enough to be counted separately.
func TestRulesMatchTheOperationsTheyName(t *testing.T) {
	rules := ProjectWorkRules()
	byName := map[string]Rule{}
	for _, r := range rules {
		byName[r.Name] = r
	}

	cases := []struct {
		rule   string
		method string
		path   string
		want   bool
	}{
		{"deploy", "POST", "/v1/deploy/pipelines/abc/trigger", true},
		{"deploy", "GET", "/v1/deploy/pipelines/abc/trigger", false},
		{"deploy", "POST", "/v1/deploy/pipelines", false},
		{"messaging", "POST", "/v1/messaging/sms", true},
		{"messaging", "GET", "/v1/messaging/topics", false},
		{"functions_exec", "POST", "/v1/functions/abc/executions", true},
	}
	for _, c := range cases {
		rule, ok := byName[c.rule]
		if !ok {
			t.Fatalf("no rule named %q", c.rule)
		}
		req := httptest.NewRequest(c.method, c.path, nil)
		if got := rule.matches(req); got != c.want {
			t.Errorf("%s %s against rule %q = %v, want %v", c.method, c.path, c.rule, got, c.want)
		}
	}
}

// A password guess is countable by the account it targets; keyed only by
// address it is not, because the address is the part an attacker changes.
func TestAccountIdentifierIsReadWithoutConsumingTheBody(t *testing.T) {
	body := `{"email":"Someone@Example.com","password":"hunter2"}`
	req := httptest.NewRequest("POST", "/v1/console/login", strings.NewReader(body))

	if got := accountIdentifier(req); got != "Someone@Example.com" {
		t.Errorf("identifier = %q", got)
	}

	// The handler still has to be able to read what it was sent.
	rest := make([]byte, len(body))
	n, _ := req.Body.Read(rest)
	if string(rest[:n]) != body {
		t.Errorf("body was consumed: got %q", string(rest[:n]))
	}
}

func TestAuthRulesCoverLoginAndRecovery(t *testing.T) {
	var ip, account bool
	for _, r := range AuthRules() {
		req := httptest.NewRequest("POST", "/v1/console/login", nil)
		if !r.matches(req) {
			continue
		}
		if r.Scope == ScopeIP {
			ip = true
		}
		if r.Scope == ScopeAccount {
			account = true
		}
	}
	if !ip || !account {
		t.Errorf("login is limited by ip=%v account=%v; both are needed", ip, account)
	}

	// A read of the same area must not be caught by a credential rule.
	for _, r := range AuthRules() {
		if r.matches(httptest.NewRequest(http.MethodGet, "/v1/console/me", nil)) {
			t.Errorf("rule %q catches a plain read", r.Name)
		}
	}
}
