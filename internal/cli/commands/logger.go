package commands

import (
	"log/slog"
	"os"

	"github.com/aki/amux/internal/core/logger"
	"github.com/spf13/cobra"
)

// Global flags for logging configuration
var (
	flagLogLevel  string
	flagLogFormat string
)

// RegisterLoggerFlags registers global logging flags
func RegisterLoggerFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&flagLogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	cmd.PersistentFlags().StringVar(&flagLogFormat, "log-format", "text", "Log format (text, json)")
}

// CreateLogger creates a logger based on CLI flags
func CreateLogger() logger.Logger {
	// Parse log level
	var level slog.Level
	switch flagLogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Parse format
	var format logger.Format
	switch flagLogFormat {
	case "json":
		format = logger.FormatJSON
	default:
		format = logger.FormatText
	}

	// Create logger
	return logger.New(
		logger.WithLevel(level),
		logger.WithFormat(format),
		logger.WithOutput(os.Stderr),
	)
}

// CreateQuietLogger creates a logger that only shows warnings and errors
func CreateQuietLogger() logger.Logger {
	return logger.New(
		logger.WithQuiet(),
		logger.WithFormat(logger.FormatText),
		logger.WithOutput(os.Stderr),
	)
}
