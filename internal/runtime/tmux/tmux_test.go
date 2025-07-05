package tmux

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/runtime"
)

// Unit tests for tmux runtime that don't require actual tmux execution

func TestTmuxRuntime_Type(t *testing.T) {
	tmpDir := t.TempDir()
	r, err := New(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "tmux", r.Type())
}

func TestTmuxRuntime_Validate(t *testing.T) {
	// When tmux is not available
	if _, err := exec.LookPath("tmux"); err != nil {
		tmpDir := t.TempDir()
		r, err := New(tmpDir)
		require.NoError(t, err)

		// Should return error
		err = r.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tmux not found")
	}
}

func TestOptions_RuntimeInterface(t *testing.T) {
	// Ensure Options implements RuntimeOptions
	var _ runtime.RuntimeOptions = Options{}
}

func TestTmuxRuntime_NotAvailable(t *testing.T) {
	// Test with non-existent tmux path
	r := &Runtime{
		executable: "/non/existent/tmux",
		baseDir:    t.TempDir(),
	}

	err := r.Validate()
	assert.Error(t, err)
}
