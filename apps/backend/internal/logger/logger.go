// Package logger provides a structured slog-based logger for Applad.
// JSON format in production, human-readable text in development.
package logger

import (
	"context"
	"log/slog"
	"os"
)

type contextKey struct{}

// New returns a *slog.Logger configured for the given environment.
func New(appEnv string) *slog.Logger {
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if appEnv != "production" {
		opts.Level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

// WithContext returns ctx with l stored inside.
func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext returns the logger stored in ctx, falling back to the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(contextKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}
