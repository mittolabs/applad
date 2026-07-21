package deploy

import (
	"regexp"
	"strconv"
	"time"
)

/*
 * Access log parsing.
 *
 * The container writes nginx's combined format; the console shows a table of
 * requests. Handing it raw lines filled every column with a placeholder, which
 * looked like the site had served nothing at all.
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

// ParseAccessLog turns log lines into entries, newest last as written.
func ParseAccessLog(lines []string, idPrefix string) []AccessEntry {
	out := make([]AccessEntry, 0, len(lines))
	for i, line := range lines {
		e := AccessEntry{ID: idPrefix + strconv.Itoa(i), Raw: line}

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
