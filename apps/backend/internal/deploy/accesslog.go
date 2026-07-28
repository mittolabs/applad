package deploy

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/*
 * Access log parsing.
 *
 * A deployed site container serves through Caddy, which writes structured JSON
 * access logs; older nginx-served containers wrote the combined text format. The
 * console shows a table of requests, so each line is turned into columns. When
 * the format went unrecognised every column showed a placeholder and only the
 * raw line was legible, which looked like the site had served nothing.
 */

// combined matches nginx's default log line:
//
//	1.2.3.4 - - [21/Jul/2026:12:28:36 +0000] "GET /about.html HTTP/1.1" 200 11199 "-" "curl/8.7"
var combined = regexp.MustCompile(
	`^(\S+) \S+ \S+ \[([^\]]+)\] "(\S+) (\S+)[^"]*" (\d{3}) (\d+|-)(?: "([^"]*)" "([^"]*)")?`)

// requestTime matches the timing our generated nginx config appends. Without
// it every request reported a duration of zero.
var requestTime = regexp.MustCompile(`rt=([0-9.]+)`)

// AccessEntry is one request, in the shape the console renders.
type AccessEntry struct {
	ID         string `json:"$id"`
	IP         string `json:"ip"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	StatusCode int    `json:"statusCode"`
	Bytes      int64  `json:"bytes"`
	UserAgent  string `json:"userAgent,omitempty"`
	// Duration is in milliseconds, as the console renders it.
	Duration  int64     `json:"duration"`
	Referer   string    `json:"referer,omitempty"`
	CreatedAt time.Time `json:"$createdAt"`
	// Raw is kept for anything the pattern does not understand, so a line is
	// never silently dropped.
	Raw string `json:"raw,omitempty"`
}

// caddyAccess is the subset of Caddy's JSON access log we render. Caddy logs one
// JSON object per request under logger "http.log.access…", with the request
// details nested and status/size/duration at the top level.
type caddyAccess struct {
	TS       float64 `json:"ts"`
	Msg      string  `json:"msg"`
	Status   int     `json:"status"`
	Size     int64   `json:"size"`
	Duration float64 `json:"duration"` // seconds
	Request  struct {
		ClientIP string              `json:"client_ip"`
		RemoteIP string              `json:"remote_ip"`
		Method   string              `json:"method"`
		URI      string              `json:"uri"`
		Headers  map[string][]string `json:"headers"`
	} `json:"request"`
}

// parseCaddyAccess reads one Caddy JSON access line. The second return is false
// for a line that is not a request (Caddy also logs startup and errors as JSON).
func parseCaddyAccess(line string) (AccessEntry, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "{") {
		return AccessEntry{}, false
	}
	var c caddyAccess
	if json.Unmarshal([]byte(line), &c) != nil || c.Request.Method == "" {
		return AccessEntry{}, false
	}
	e := AccessEntry{
		IP:         firstNonEmptyStr(c.Request.ClientIP, c.Request.RemoteIP),
		Method:     c.Request.Method,
		Path:       c.Request.URI,
		StatusCode: c.Status,
		Bytes:      c.Size,
		Duration:   int64(c.Duration * 1000),
	}
	if ua := c.Request.Headers["User-Agent"]; len(ua) > 0 {
		e.UserAgent = ua[0]
	}
	if rf := c.Request.Headers["Referer"]; len(rf) > 0 {
		e.Referer = rf[0]
	}
	if c.TS > 0 {
		sec := int64(c.TS)
		e.CreatedAt = time.Unix(sec, int64((c.TS-float64(sec))*1e9)).UTC()
	}
	return e, true
}

func firstNonEmptyStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// ParseAccessLog turns log lines into entries, newest last as written.
func ParseAccessLog(lines []string, idPrefix string) []AccessEntry {
	out := make([]AccessEntry, 0, len(lines))
	for i, line := range lines {
		e := AccessEntry{ID: idPrefix + strconv.Itoa(i), Raw: line}

		// Caddy JSON first (what deployed sites write today), then the older
		// nginx combined format.
		if caddy, ok := parseCaddyAccess(line); ok {
			caddy.ID = e.ID
			out = append(out, caddy)
			continue
		}

		m := combined.FindStringSubmatch(line)
		if m == nil {
			// nginx also logs errors and startup notices, which have no
			// request in them; they are still worth showing.
			out = append(out, e)
			continue
		}

		e.IP, e.Method, e.Path = m[1], m[3], m[4]
		e.StatusCode, _ = strconv.Atoi(m[5])
		if m[6] != "-" {
			e.Bytes, _ = strconv.ParseInt(m[6], 10, 64)
		}
		e.Referer, e.UserAgent = m[7], m[8]
		if rt := requestTime.FindStringSubmatch(line); rt != nil {
			if secs, err := strconv.ParseFloat(rt[1], 64); err == nil {
				e.Duration = int64(secs * 1000)
			}
		}
		if t, err := time.Parse("02/Jan/2006:15:04:05 -0700", m[2]); err == nil {
			e.CreatedAt = t
		}
		e.Raw = ""
		out = append(out, e)
	}
	return out
}
