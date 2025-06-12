package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("creates logger with default options", func(t *testing.T) {
		logger := New()
		if logger == nil {
			t.Fatal("expected logger, got nil")
		}
	})

	t.Run("creates logger with custom options", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(
			WithOutput(&buf),
			WithLevel(slog.LevelDebug),
			WithFormat(FormatText),
		)

		logger.Debug("test message", "key", "value")
		output := buf.String()

		if !strings.Contains(output, "test message") {
			t.Errorf("expected output to contain 'test message', got: %s", output)
		}
		if !strings.Contains(output, "key=value") {
			t.Errorf("expected output to contain 'key=value', got: %s", output)
		}
	})

	t.Run("respects log level", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(
			WithOutput(&buf),
			WithLevel(slog.LevelWarn),
		)

		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")

		output := buf.String()
		if strings.Contains(output, "debug message") {
			t.Error("debug message should not appear with warn level")
		}
		if strings.Contains(output, "info message") {
			t.Error("info message should not appear with warn level")
		}
		if !strings.Contains(output, "warn message") {
			t.Error("warn message should appear with warn level")
		}
		if !strings.Contains(output, "error message") {
			t.Error("error message should appear with warn level")
		}
	})
}

func TestNop(t *testing.T) {
	logger := Nop()
	if logger == nil {
		t.Fatal("expected logger, got nil")
	}

	// Nop logger should not panic when called
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")
}

func TestDefault(t *testing.T) {
	logger := Default()
	if logger == nil {
		t.Fatal("expected logger, got nil")
	}
}

func TestLoggerMethods(t *testing.T) {
	var buf bytes.Buffer
	logger := New(
		WithOutput(&buf),
		WithLevel(slog.LevelDebug),
		WithFormat(FormatText),
	)

	t.Run("Debug", func(t *testing.T) {
		buf.Reset()
		logger.Debug("debug message", "key", "value")
		if !strings.Contains(buf.String(), "debug message") {
			t.Error("expected debug message in output")
		}
	})

	t.Run("Info", func(t *testing.T) {
		buf.Reset()
		logger.Info("info message", "key", "value")
		if !strings.Contains(buf.String(), "info message") {
			t.Error("expected info message in output")
		}
	})

	t.Run("Warn", func(t *testing.T) {
		buf.Reset()
		logger.Warn("warn message", "key", "value")
		if !strings.Contains(buf.String(), "warn message") {
			t.Error("expected warn message in output")
		}
	})

	t.Run("Error", func(t *testing.T) {
		buf.Reset()
		logger.Error("error message", "key", "value")
		if !strings.Contains(buf.String(), "error message") {
			t.Error("expected error message in output")
		}
	})
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer
	logger := New(
		WithOutput(&buf),
		WithLevel(slog.LevelDebug),
		WithFormat(FormatText),
	)

	// Create logger with additional context
	contextLogger := logger.With("request_id", "123")
	contextLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "request_id=123") {
		t.Errorf("expected output to contain 'request_id=123', got: %s", output)
	}
}

func TestWithGroup(t *testing.T) {
	var buf bytes.Buffer
	logger := New(
		WithOutput(&buf),
		WithLevel(slog.LevelDebug),
		WithFormat(FormatJSON),
	)

	// Create logger with group
	groupLogger := logger.WithGroup("http")
	groupLogger.Info("request", "method", "GET", "path", "/api")

	output := buf.String()
	if !strings.Contains(output, `"http":{`) {
		t.Errorf("expected output to contain http group, got: %s", output)
	}
}

func TestContext(t *testing.T) {
	t.Run("WithContext and FromContext", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(WithOutput(&buf))

		ctx := context.Background()
		ctx = WithContext(ctx, logger)

		retrievedLogger := FromContext(ctx)
		retrievedLogger.Info("test message")

		if !strings.Contains(buf.String(), "test message") {
			t.Error("expected message from context logger")
		}
	})

	t.Run("FromContext returns Nop when no logger", func(t *testing.T) {
		ctx := context.Background()
		logger := FromContext(ctx)

		// Should not panic
		logger.Info("test message")
	})
}

func TestFormats(t *testing.T) {
	t.Run("JSON format", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(
			WithOutput(&buf),
			WithFormat(FormatJSON),
		)

		logger.Info("test message", "key", "value")
		output := buf.String()

		if !strings.Contains(output, `"msg":"test message"`) {
			t.Errorf("expected JSON format, got: %s", output)
		}
		if !strings.Contains(output, `"key":"value"`) {
			t.Errorf("expected JSON format with key-value, got: %s", output)
		}
	})

	t.Run("Text format", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(
			WithOutput(&buf),
			WithFormat(FormatText),
		)

		logger.Info("test message", "key", "value")
		output := buf.String()

		if !strings.Contains(output, "test message") {
			t.Errorf("expected text format, got: %s", output)
		}
		if !strings.Contains(output, "key=value") {
			t.Errorf("expected text format with key=value, got: %s", output)
		}
	})
}

func TestOptions(t *testing.T) {
	t.Run("WithDebug", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(
			WithOutput(&buf),
			WithDebug(),
		)

		logger.Debug("debug message")
		if !strings.Contains(buf.String(), "debug message") {
			t.Error("WithDebug should enable debug logging")
		}
	})

	t.Run("WithQuiet", func(t *testing.T) {
		var buf bytes.Buffer
		logger := New(
			WithOutput(&buf),
			WithQuiet(),
		)

		logger.Info("info message")
		logger.Warn("warn message")

		output := buf.String()
		if strings.Contains(output, "info message") {
			t.Error("WithQuiet should suppress info messages")
		}
		if !strings.Contains(output, "warn message") {
			t.Error("WithQuiet should allow warn messages")
		}
	})
}
