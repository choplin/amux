package commands

import (
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Global flags for logging configuration
var (
	flagLogLevel  string
	flagLogFormat string
	flagDebug     bool
)

// RegisterLoggerFlags registers global logging flags
func RegisterLoggerFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&flagLogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	cmd.PersistentFlags().StringVar(&flagLogFormat, "log-format", "text", "Log format (text, json)")
	cmd.PersistentFlags().BoolVar(&flagDebug, "debug", false, "Enable debug logging (shortcut for --log-level debug)")
}

// InitializeSlog initializes the global slog logger based on CLI flags
func InitializeSlog() {
	// Debug flag overrides log level
	levelStr := flagLogLevel
	if flagDebug {
		levelStr = "debug"
	}

	// Parse log level
	var level slog.Level
	switch strings.ToLower(levelStr) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler based on format
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	switch strings.ToLower(flagLogFormat) {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, opts)
	default:
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	// Set as default logger
	slog.SetDefault(slog.New(handler))
}
