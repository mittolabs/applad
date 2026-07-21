package deploy

import "testing"

// The console renders a table of requests, so a raw line leaves every column
// empty — which reads as a site that has served nothing.
func TestParsesNginxCombinedFormat(t *testing.T) {
	lines := []string{
		`172.18.0.21 - - [21/Jul/2026:12:28:36 +0000] "GET /about.html HTTP/1.1" 200 11199 "-" "curl/8.7.1"`,
	}
	got := ParseAccessLog(lines, "log-")
	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1", len(got))
	}

	e := got[0]
	if e.Method != "GET" || e.Path != "/about.html" || e.StatusCode != 200 || e.Bytes != 11199 {
		t.Errorf("parsed = %+v", e)
	}
	if e.UserAgent != "curl/8.7.1" {
		t.Errorf("user agent = %q", e.UserAgent)
	}
	if e.CreatedAt.IsZero() {
		t.Error("timestamp was not parsed, so the table cannot order by it")
	}
}

// nginx also writes startup notices and errors with no request in them. They
// are still worth showing rather than dropping silently.
func TestKeepsLinesItCannotParse(t *testing.T) {
	got := ParseAccessLog([]string{"2026/07/21 12:00:00 [notice] 1#1: start worker process"}, "log-")
	if len(got) != 1 {
		t.Fatalf("got %d entries, want the line kept", len(got))
	}
	if got[0].Raw == "" {
		t.Error("an unparsed line must keep its text")
	}
}

func TestHandlesAMissingByteCount(t *testing.T) {
	got := ParseAccessLog([]string{
		`10.0.0.1 - - [21/Jul/2026:12:00:00 +0000] "HEAD / HTTP/1.1" 304 -`,
	}, "log-")
	if got[0].StatusCode != 304 || got[0].Bytes != 0 {
		t.Errorf("parsed = %+v", got[0])
	}
}
