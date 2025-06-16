package session

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendInputCmd(t *testing.T) {
	cmd := sendInputCmd()

	t.Run("command properties", func(t *testing.T) {
		assert.Equal(t, "send-input", cmd.Use[:10])
		assert.Contains(t, cmd.Use, "<session-id>")
		assert.Contains(t, cmd.Use, "<input-text>")
		assert.Equal(t, []string{"send"}, cmd.Aliases)
		assert.NotEmpty(t, cmd.Short)
		assert.NotEmpty(t, cmd.Long)
		assert.Contains(t, cmd.Long, "Examples:")
	})

	t.Run("requires exactly 2 arguments", func(t *testing.T) {
		err := cmd.Args(nil, []string{})
		assert.Error(t, err)

		err = cmd.Args(nil, []string{"session-id"})
		assert.Error(t, err)

		err = cmd.Args(nil, []string{"session-id", "input"})
		assert.NoError(t, err)

		err = cmd.Args(nil, []string{"session-id", "input", "extra"})
		assert.Error(t, err)
	})

	t.Run("has RunE function", func(t *testing.T) {
		require.NotNil(t, cmd.RunE)
	})
}

func TestSendInputToSession(t *testing.T) {
	// Note: Full integration tests would require mocking the session manager
	// and session interfaces. For now, we're just testing that the command
	// is properly constructed and would fail appropriately when run outside
	// an initialized project.

	t.Run("fails with invalid session", func(t *testing.T) {
		cmd := sendInputCmd()
		// Set a context to avoid nil context panic
		cmd.SetContext(context.Background())
		// Try to send input to a non-existent session
		err := cmd.RunE(cmd, []string{"non-existent-session", "test input"})
		assert.Error(t, err)
		// Could fail either at project root detection or session lookup
		assert.True(t,
			strings.Contains(err.Error(), "project root") ||
				strings.Contains(err.Error(), "session not found"),
			"Expected error about project root or session not found, got: %v", err)
	})
}
