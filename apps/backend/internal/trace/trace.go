// Package trace implements W3C Trace Context propagation (traceparent header).
// It generates/propagates trace+span IDs across HTTP requests and stores them
// in context. No external dependencies — uses only crypto/rand + encoding/hex.
package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
)

type contextKey struct{}

// Span holds the trace and span IDs for a single request.
type Span struct {
	TraceID string
	SpanID  string
}

// WithSpan stores a Span in ctx.
func WithSpan(ctx context.Context, s Span) context.Context {
	return context.WithValue(ctx, contextKey{}, s)
}

// FromContext returns the Span stored in ctx (zero value if absent).
func FromContext(ctx context.Context) Span {
	s, _ := ctx.Value(contextKey{}).(Span)
	return s
}

// newTraceID generates a random 16-byte (128-bit) hex trace ID.
func newTraceID() string {
	b := make([]byte, 16)
	rand.Read(b) //nolint:errcheck
	return hex.EncodeToString(b)
}

// newSpanID generates a random 8-byte (64-bit) hex span ID.
func newSpanID() string {
	b := make([]byte, 8)
	rand.Read(b) //nolint:errcheck
	return hex.EncodeToString(b)
}

// parse extracts trace and span IDs from a traceparent header value.
// Format: 00-<traceId>-<parentSpanId>-<flags>
func parse(header string) (traceID, spanID string, ok bool) {
	parts := strings.Split(header, "-")
	if len(parts) != 4 || parts[0] != "00" {
		return "", "", false
	}
	if len(parts[1]) != 32 || len(parts[2]) != 16 {
		return "", "", false
	}
	return parts[1], parts[2], true
}

// Middleware is a chi-compatible middleware that reads or generates a W3C
// traceparent, stores the Span in context, and enriches the request logger.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var span Span

		if tp := r.Header.Get("traceparent"); tp != "" {
			if traceID, spanID, ok := parse(tp); ok {
				span = Span{TraceID: traceID, SpanID: newSpanID()}
				// Propagate parent span ID for debugging
				_ = spanID
			}
		}
		if span.TraceID == "" {
			span = Span{TraceID: newTraceID(), SpanID: newSpanID()}
		}

		// Set traceparent on the response so callers can correlate
		w.Header().Set("traceparent", "00-"+span.TraceID+"-"+span.SpanID+"-01")

		ctx := WithSpan(r.Context(), span)

		// Enrich context logger with trace fields
		if l, ok := r.Context().Value(struct{ loggerKey string }{"logger"}).(*slog.Logger); ok {
			ctx = context.WithValue(ctx,
				struct{ loggerKey string }{"logger"},
				l.With("trace_id", span.TraceID, "span_id", span.SpanID),
			)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// TraceID returns the trace ID from ctx, or empty string.
func TraceID(ctx context.Context) string {
	return FromContext(ctx).TraceID
}
