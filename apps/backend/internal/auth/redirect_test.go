package auth

import "testing"

// The OAuth redirect is chosen by an unauthenticated caller, so every one of
// these is a real attack shape rather than a hypothetical.
func TestSafeRedirectTarget(t *testing.T) {
	registered := []string{
		"funnier://auth",
		"https://funnier.app/callback",
		"http://localhost:3000/cb",
	}

	allowed := []string{
		"",                                  // becomes "/"
		"/",                                 // relative
		"/projects?created=1",               // relative with a query
		"funnier://auth",                    // exact custom scheme
		"funnier://auth?secret=x",           // query does not change the target
		"FUNNIER://AUTH",                    // scheme and host are case-insensitive
		"https://funnier.app/callback",      // exact
		"https://funnier.app/callback/done", // under it, at a boundary
		"http://localhost:3000/cb",          // port matches
	}
	for _, in := range allowed {
		got := safeRedirectTarget(in, registered)
		want := in
		if in == "" {
			want = "/"
		}
		if got != want {
			t.Errorf("safeRedirectTarget(%q) = %q, want %q", in, got, want)
		}
	}

	refused := map[string]string{
		"https://evil.com/":                       "unregistered host",
		"//evil.com":                              "protocol-relative, reads as a path",
		"funnier://auth.evil.com":                 "host suffix, not the registered host",
		"funnierx://auth":                         "scheme is not a prefix match",
		"https://funnier.app.evil.com/callback":   "host is not a prefix match",
		"https://funnier.app/callbackx":           "path must break at a segment",
		"https://funnier.app/other":               "sibling path",
		"http://funnier.app/callback":             "scheme downgrade",
		"http://localhost:4000/cb":                "different port is a different app",
		"https://user@funnier.app/callback":       "credentials in the authority",
		"https://funnier.app/callback/../../root": "climbing out of the registered path",
	}
	for in, why := range refused {
		if got := safeRedirectTarget(in, registered); got != "/" {
			t.Errorf("safeRedirectTarget(%q) = %q, want \"/\" (%s)", in, got, why)
		}
	}

	// With nothing registered, only relative paths survive — the behaviour
	// every existing project keeps.
	if got := safeRedirectTarget("funnier://auth", nil); got != "/" {
		t.Errorf("unregistered project allowed %q", got)
	}
	if got := safeRedirectTarget("/projects", nil); got != "/projects" {
		t.Errorf("relative path refused: %q", got)
	}
}

func TestIsAbsoluteRedirectAndWithParam(t *testing.T) {
	if isAbsoluteRedirect("/projects") {
		t.Error("a relative path is not absolute")
	}
	if !isAbsoluteRedirect("funnier://auth") {
		t.Error("a custom scheme is absolute")
	}
	if !isAbsoluteRedirect("https://funnier.app/cb") {
		t.Error("an https URL is absolute")
	}

	got := withParam("funnier://auth", "secret", "abc/123")
	if got != "funnier://auth?secret=abc%2F123" {
		t.Errorf("withParam = %q", got)
	}
	// An existing query is kept, not replaced.
	got = withParam("https://funnier.app/cb?state=1", "secret", "s")
	if got != "https://funnier.app/cb?secret=s&state=1" {
		t.Errorf("withParam kept/added wrongly: %q", got)
	}
}
