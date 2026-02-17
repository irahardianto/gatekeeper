package logger

import (
	"context"
	"log/slog"
	"os"
)

type contextKey struct{}

var loggerKey = contextKey{}

// New creates a new structured logger.
// If verbose is true, the log level is set to Debug.
// If json is true, the output format is JSON.
func New(verbose, json bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if json {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// WithContext returns a new context with the given logger attached.
func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext returns the logger from the context.
// If no logger is found, it returns the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}
