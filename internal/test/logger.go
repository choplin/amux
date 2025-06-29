// Package test provides common test utilities for the amux project.
package test

import (
	"log/slog"
	"os"
	"sync"
)

var initOnce sync.Once

// InitTestLogger initializes the logger for tests with warn level to match CLI default
func InitTestLogger() {
	initOnce.Do(func() {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})))
	})
}
