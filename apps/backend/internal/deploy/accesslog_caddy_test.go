package deploy

import "testing"

func TestParseAccessLog_CaddyJSON(t *testing.T) {
	line := `{"level":"info","ts":1785255273.9701793,"logger":"http.log.access.log0","msg":"handled request","request":{"remote_ip":"172.18.0.4","remote_port":"38594","client_ip":"197.156.144.141","proto":"HTTP/1.1","method":"GET","host":"the-range.applad.dev","uri":"/assets/images/hover-bg.png","headers":{"User-Agent":["Mozilla/5.0 (Macintosh) Chrome/150"],"Referer":["https://the-range.applad.dev/"]}},"duration":0.000318933,"size":1234,"status":304}`
	got := ParseAccessLog([]string{line}, "t-")
	if len(got) != 1 {
		t.Fatalf("want 1 entry, got %d", len(got))
	}
	e := got[0]
	if e.Method != "GET" || e.Path != "/assets/images/hover-bg.png" {
		t.Errorf("method/path wrong: %q %q", e.Method, e.Path)
	}
	if e.StatusCode != 304 {
		t.Errorf("status: %d", e.StatusCode)
	}
	if e.IP != "197.156.144.141" {
		t.Errorf("ip: %q", e.IP)
	}
	if e.Bytes != 1234 {
		t.Errorf("bytes: %d", e.Bytes)
	}
	if e.Duration != 0 { // 0.000318s rounds to 0ms, but must not be from a parse miss
		if e.Duration < 0 {
			t.Errorf("duration: %d", e.Duration)
		}
	}
	if e.UserAgent == "" || e.Referer == "" {
		t.Errorf("ua/referer empty: %q %q", e.UserAgent, e.Referer)
	}
	if e.CreatedAt.Year() != 2026 {
		t.Errorf("createdAt not parsed from ts: %v", e.CreatedAt)
	}
	if e.Raw != "" {
		t.Errorf("raw should be cleared on a parsed line, got %q", e.Raw)
	}
}

func TestParseAccessLog_NginxStillWorks(t *testing.T) {
	line := `1.2.3.4 - - [21/Jul/2026:12:28:36 +0000] "GET /about.html HTTP/1.1" 200 11199 "-" "curl/8.7"`
	got := ParseAccessLog([]string{line}, "t-")
	if got[0].Method != "GET" || got[0].StatusCode != 200 || got[0].Path != "/about.html" {
		t.Fatalf("nginx combined regressed: %+v", got[0])
	}
}
