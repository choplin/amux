package local

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Unit tests for local runtime that don't require actual process execution

func TestLocalRuntime_Type(t *testing.T) {
	r := New()
	assert.Equal(t, "local", r.Type())
}

func TestDetachedRuntime_Type(t *testing.T) {
	r := NewDetachedRuntime()
	assert.Equal(t, "local-detached", r.Type())
}

func TestLocalRuntime_Validate(t *testing.T) {
	r := New()
	// Local runtime should always be valid
	assert.NoError(t, r.Validate())
}

func TestDetachedRuntime_Validate(t *testing.T) {
	r := NewDetachedRuntime()
	// Detached runtime should always be valid
	assert.NoError(t, r.Validate())
}
