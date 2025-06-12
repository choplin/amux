package logger

import (
	"io"
	"log/slog"
)

// Format represents the output format for logs
type Format string

const (
	// FormatText outputs human-readable text format
	FormatText Format = "text"
	// FormatJSON outputs structured JSON format
	FormatJSON Format = "json"
)

// config holds the logger configuration
type config struct {
	level  slog.Level
	output io.Writer
	format Format
}

// Option is a function that configures a logger
type Option func(*config)

// WithLevel sets the minimum log level
func WithLevel(level slog.Level) Option {
	return func(c *config) {
		c.level = level
	}
}

// WithOutput sets the output writer for logs
func WithOutput(w io.Writer) Option {
	return func(c *config) {
		c.output = w
	}
}

// WithFormat sets the output format
func WithFormat(format Format) Option {
	return func(c *config) {
		c.format = format
	}
}

// WithDebug is a convenience option to enable debug logging
func WithDebug() Option {
	return WithLevel(slog.LevelDebug)
}

// WithQuiet is a convenience option to only show warnings and errors
func WithQuiet() Option {
	return WithLevel(slog.LevelWarn)
}
