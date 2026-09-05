package analytics

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"
)

// routeKey identifies a unique (projectID, method, path) combination.
type routeKey struct {
	projectID, method, path string
}

// sampleBucket accumulates raw latency samples and error counts for one route
// within the current flush window.
type sampleBucket struct {
	samples  []float64
	total    int64
	errors   int64
	windowMs int64 // elapsed ms since last flush (set at flush time)
}

// PerfCollector buffers per-request latency samples in memory and periodically
// flushes aggregated percentile snapshots to analytics_perf_snapshots via
// Service.RecordPerf. Call Start() once to begin the flush loop.
type PerfCollector struct {
	svc      *Service
	mu       sync.Mutex
	buckets  map[routeKey]*sampleBucket
	interval time.Duration
}

// NewPerfCollector creates a collector that flushes every interval.
// A typical value is 60 * time.Second.
func NewPerfCollector(svc *Service, interval time.Duration) *PerfCollector {
	return &PerfCollector{
		svc:      svc,
		buckets:  make(map[routeKey]*sampleBucket),
		interval: interval,
	}
}

// Record adds one latency sample for the given project/method/path.
// isError should be true when the response status code >= 400.
func (c *PerfCollector) Record(projectID, method, path string, latencyMs float64, isError bool) {
	if projectID == "" {
		return
	}
	k := routeKey{projectID, method, path}
	c.mu.Lock()
	b, ok := c.buckets[k]
	if !ok {
		b = &sampleBucket{}
		c.buckets[k] = b
	}
	b.samples = append(b.samples, latencyMs)
	b.total++
	if isError {
		b.errors++
	}
	c.mu.Unlock()
}

// Start launches the background flush goroutine. It stops when ctx is done.
func (c *PerfCollector) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()
		start := time.Now()
		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				elapsed := t.Sub(start)
				start = t
				c.flush(ctx, elapsed)
			}
		}
	}()
}

// flush aggregates all buffered samples and writes them to the DB.
func (c *PerfCollector) flush(ctx context.Context, elapsed time.Duration) {
	c.mu.Lock()
	snap := c.buckets
	c.buckets = make(map[routeKey]*sampleBucket)
	c.mu.Unlock()

	elapsedSec := elapsed.Seconds()
	for k, b := range snap {
		if len(b.samples) == 0 {
			continue
		}
		sort.Float64s(b.samples)
		n := len(b.samples)
		rps := float64(b.total) / elapsedSec
		errPct := 0.0
		if b.total > 0 {
			errPct = float64(b.errors) / float64(b.total) * 100
		}
		_ = c.svc.RecordPerf(ctx, k.projectID, RecordPerfRequest{
			Method:   k.method,
			Path:     k.path,
			P50Ms:    percentile(b.samples, n, 0.50),
			P75Ms:    percentile(b.samples, n, 0.75),
			P95Ms:    percentile(b.samples, n, 0.95),
			P99Ms:    percentile(b.samples, n, 0.99),
			RPS:      rps,
			ErrorPct: errPct,
			ReqCount: b.total,
		})
	}
}

// percentile returns the p-th percentile value (0.0–1.0) from a sorted slice.
func percentile(sorted []float64, n int, p float64) float64 {
	if n == 0 {
		return 0
	}
	idx := p * float64(n-1)
	lo := int(math.Floor(idx))
	hi := int(math.Ceil(idx))
	if lo == hi {
		return sorted[lo]
	}
	return sorted[lo] + (idx-float64(lo))*(sorted[hi]-sorted[lo])
}
