//go:build integration
// +build integration

package storage

import (
	"testing"
)

// Integration tests that need real I/O behavior
// Run with: go test -tags=integration ./internal/cli/commands/session/storage

func TestStorageIntegration(t *testing.T) {
	t.Skip("Integration tests are separate from unit tests")
}

// The existing storage_test.go tests are actually integration tests
// because they rely on real file I/O and command execution.
// For unit tests, we would need to mock the file system and command execution.
