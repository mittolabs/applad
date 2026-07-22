// Package metrics provides a zero-dependency Prometheus-format metrics store.
// Counters and histograms are kept in memory with sync/atomic and exposed
// via a plain-text HTTP handler at /metrics (Prometheus exposition format).
package metrics

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ── Counter ──────────────────────────────────────────────────────────────────

// Counter is a monotonically-increasing uint64 counter.
type Counter struct {
	v    atomic.Uint64
	name string
	help string
}

func (c *Counter) Inc()          { c.v.Add(1) }
func (c *Counter) Add(n uint64)  { c.v.Add(n) }
func (c *Counter) Value() uint64 { return c.v.Load() }

// ── LabelCounter ─────────────────────────────────────────────────────────────

// LabelCounter is a set of counters keyed by label values.
type LabelCounter struct {
	mu     sync.RWMutex
	counts map[string]*atomic.Uint64
	name   string
	help   string
	labels []string // label names in order
}

func (lc *LabelCounter) Inc(labelValues ...string) {
	key := strings.Join(labelValues, "\x00")
	lc.mu.RLock()
	v, ok := lc.counts[key]
	lc.mu.RUnlock()
	if ok {
		v.Add(1)
		return
	}
	lc.mu.Lock()
	if lc.counts[key] == nil {
		lc.counts[key] = &atomic.Uint64{}
	}
	lc.counts[key].Add(1)
	lc.mu.Unlock()
}

// ── Histogram ─────────────────────────────────────────────────────────────────

// Histogram tracks a distribution of values using fixed buckets.
type Histogram struct {
	mu      sync.Mutex
	buckets []float64
	counts  []atomic.Uint64
	sum     atomic.Uint64 // stored as microseconds for precision
	total   atomic.Uint64
	name    string
	help    string
}

// Observe records a duration in seconds.
func (h *Histogram) Observe(seconds float64) {
	us := uint64(seconds * 1e6)
	h.sum.Add(us)
	h.total.Add(1)
	for i, bound := range h.buckets {
		if seconds <= bound {
			h.counts[i].Add(1)
		}
	}
}

// ── Registry ─────────────────────────────────────────────────────────────────

// Registry holds all registered metrics.
type Registry struct {
	mu         sync.RWMutex
	counters   []*Counter
	lCounters  []*LabelCounter
	histograms []*Histogram
}

var Default = &Registry{}

// Counter registers and returns a counter.
func (r *Registry) Counter(name, help string) *Counter {
	c := &Counter{name: name, help: help}
	r.mu.Lock()
	r.counters = append(r.counters, c)
	r.mu.Unlock()
	return c
}

// LabelCounter registers and returns a labelled counter.
func (r *Registry) LabelCounter(name, help string, labels ...string) *LabelCounter {
	lc := &LabelCounter{name: name, help: help, labels: labels, counts: map[string]*atomic.Uint64{}}
	r.mu.Lock()
	r.lCounters = append(r.lCounters, lc)
	r.mu.Unlock()
	return lc
}

// Histogram registers and returns a histogram with given bucket boundaries (seconds).
func (r *Registry) Histogram(name, help string, buckets []float64) *Histogram {
	sorted := make([]float64, len(buckets))
	copy(sorted, buckets)
	sort.Float64s(sorted)
	h := &Histogram{
		name:    name,
		help:    help,
		buckets: sorted,
		counts:  make([]atomic.Uint64, len(sorted)),
	}
	r.mu.Lock()
	r.histograms = append(r.histograms, h)
	r.mu.Unlock()
	return h
}

// writeText writes Prometheus exposition format to w.
func (r *Registry) writeText(w io.Writer) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, c := range r.counters {
		fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n%s %d\n",
			c.name, c.help, c.name, c.name, c.Value())
	}

	for _, lc := range r.lCounters {
		fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n", lc.name, lc.help, lc.name)
		lc.mu.RLock()
		for key, v := range lc.counts {
			vals := strings.Split(key, "\x00")
			var pairs []string
			for i, lname := range lc.labels {
				val := ""
				if i < len(vals) {
					val = vals[i]
				}
				pairs = append(pairs, fmt.Sprintf(`%s=%q`, lname, val))
			}
			fmt.Fprintf(w, "%s{%s} %d\n", lc.name, strings.Join(pairs, ","), v.Load())
		}
		lc.mu.RUnlock()
	}

	for _, h := range r.histograms {
		fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s histogram\n", h.name, h.help, h.name)
		cumulative := uint64(0)
		for i, bound := range h.buckets {
			cumulative += h.counts[i].Load()
			fmt.Fprintf(w, "%s_bucket{le=\"%g\"} %d\n", h.name, bound, cumulative)
		}
		fmt.Fprintf(w, "%s_bucket{le=\"+Inf\"} %d\n", h.name, h.total.Load())
		fmt.Fprintf(w, "%s_sum %g\n", h.name, float64(h.sum.Load())/1e6)
		fmt.Fprintf(w, "%s_count %d\n", h.name, h.total.Load())
	}
}

// Handler returns an http.Handler that serves metrics in Prometheus format.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		r.writeText(w)
	})
}

// ── Built-in application metrics ─────────────────────────────────────────────

var (
	// HTTPRequests counts requests by method, path pattern, and status class.
	HTTPRequests = Default.LabelCounter(
		"http_requests_total",
		"Total HTTP requests.",
		"method", "path", "status",
	)
	// HTTPDuration tracks request latency in seconds.
	HTTPDuration = Default.Histogram(
		"http_request_duration_seconds",
		"HTTP request duration in seconds.",
		[]float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	)
	// QueueJobs counts queue job outcomes by worker and status.
	QueueJobs = Default.LabelCounter(
		"queue_jobs_total",
		"Total queue jobs processed.",
		"worker", "status",
	)
	// DBErrors counts database errors by operation.
	DBErrors = Default.Counter(
		"db_errors_total",
		"Total database errors.",
	)
	// ActiveWebSocketConns tracks current WebSocket connections.
	ActiveWebSocketConns = Default.Counter(
		"websocket_connections_active",
		"Current active WebSocket connections.",
	)
)

// ObserveRequest records an HTTP request's method, route pattern, status code,
// and duration. Call after the response is written.
func ObserveRequest(method, pattern string, status int, start time.Time) {
	statusClass := fmt.Sprintf("%dxx", status/100)
	HTTPRequests.Inc(method, pattern, statusClass)
	HTTPDuration.Observe(time.Since(start).Seconds())
}
