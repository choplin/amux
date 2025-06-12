// Package logger provides structured logging functionality for amux.
// It wraps Go's slog package to provide a consistent logging interface
// with support for different log levels, formats, and outputs.
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// Logger is the interface for structured logging in amux.
// It provides methods for different log levels and supports structured fields.
type Logger interface {
	// Debug logs a debug message with optional key-value pairs
	Debug(msg string, args ...any)
	// Info logs an info message with optional key-value pairs
	Info(msg string, args ...any)
	// Warn logs a warning message with optional key-value pairs
	Warn(msg string, args ...any)
	// Error logs an error message with optional key-value pairs
	Error(msg string, args ...any)
	// With returns a new Logger with additional context fields
	With(args ...any) Logger
	// WithGroup returns a new Logger with a group prefix
	WithGroup(name string) Logger
}

// slogLogger wraps slog.Logger to implement our Logger interface
type slogLogger struct {
	logger *slog.Logger
}

// New creates a new Logger with the specified configuration
func New(opts ...Option) Logger {
	cfg := &config{
		level:  slog.LevelInfo,
		output: os.Stderr,
		format: FormatText,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	var handler slog.Handler
	handlerOpts := &slog.HandlerOptions{
		Level: cfg.level,
	}

	switch cfg.format {
	case FormatJSON:
		handler = slog.NewJSONHandler(cfg.output, handlerOpts)
	case FormatText:
		handler = slog.NewTextHandler(cfg.output, handlerOpts)
	default:
		handler = slog.NewTextHandler(cfg.output, handlerOpts)
	}

	return &slogLogger{
		logger: slog.New(handler),
	}
}

// Nop returns a no-op logger that discards all log messages
func Nop() Logger {
	return &slogLogger{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// Default returns a default logger that writes to stderr
func Default() Logger {
	return New()
}

// Debug implements Logger
func (l *slogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info implements Logger
func (l *slogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn implements Logger
func (l *slogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error implements Logger
func (l *slogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// With implements Logger
func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{
		logger: l.logger.With(args...),
	}
}

// WithGroup implements Logger
func (l *slogLogger) WithGroup(name string) Logger {
	return &slogLogger{
		logger: l.logger.WithGroup(name),
	}
}

// FromContext returns a Logger from the context, or a no-op logger if not found
func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerKey{}).(Logger); ok {
		return logger
	}
	return Nop()
}

// WithContext returns a new context with the logger attached
func WithContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

type loggerKey struct{}
