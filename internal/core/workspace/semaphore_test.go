package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSessionChecker implements SessionChecker for testing
type mockSessionChecker struct {
	activeSessions map[string]bool
}

func (m *mockSessionChecker) IsSessionActive(sessionID string) (bool, error) {
	active, exists := m.activeSessions[sessionID]
	return exists && active, nil
}

func TestSemaphoreManager_AcquireRelease(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "workspaces")

	checker := &mockSessionChecker{
		activeSessions: map[string]bool{
			"session-1": true,
		},
	}

	sm := NewSemaphoreManager(basePath, checker)

	workspaceID := "ws-test-123"
	holder := Holder{
		ID:          "session-1",
		Type:        HolderTypeSession,
		SessionID:   "session-1",
		Description: "Test session",
	}

	// Test acquire
	err := sm.Acquire(workspaceID, holder)
	require.NoError(t, err)

	// Verify holder was added
	holders, err := sm.GetHolders(workspaceID)
	require.NoError(t, err)
	assert.Len(t, holders, 1)
	assert.Equal(t, "session-1", holders[0].ID)
	assert.Equal(t, workspaceID, holders[0].WorkspaceID)
	assert.False(t, holders[0].Timestamp.IsZero())

	// Test release
	err = sm.Release(workspaceID, "session-1")
	require.NoError(t, err)

	// Verify holder was removed
	holders, err = sm.GetHolders(workspaceID)
	require.NoError(t, err)
	assert.Empty(t, holders)
}

func TestSemaphoreManager_MultipleHolders(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "workspaces")

	checker := &mockSessionChecker{
		activeSessions: map[string]bool{
			"session-1": true,
			"session-2": true,
		},
	}

	sm := NewSemaphoreManager(basePath, checker)
	workspaceID := "ws-test-123"

	// Add multiple holders
	holders := []Holder{
		{ID: "session-1", Type: HolderTypeSession, SessionID: "session-1"},
		{ID: "session-2", Type: HolderTypeSession, SessionID: "session-2"},
		{ID: "cli-1", Type: HolderTypeCLI, Description: "CLI command"},
	}

	for _, h := range holders {
		err := sm.Acquire(workspaceID, h)
		require.NoError(t, err)
	}

	// Verify all holders
	gotHolders, err := sm.GetHolders(workspaceID)
	require.NoError(t, err)
	assert.Len(t, gotHolders, 3)

	// Test IsInUse
	inUse, usageHolders, err := sm.IsInUse(workspaceID)
	require.NoError(t, err)
	assert.True(t, inUse)
	assert.Len(t, usageHolders, 3)

	// Release one holder
	err = sm.Release(workspaceID, "session-1")
	require.NoError(t, err)

	// Verify remaining holders
	gotHolders, err = sm.GetHolders(workspaceID)
	require.NoError(t, err)
	assert.Len(t, gotHolders, 2)
}

func TestSemaphoreManager_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "workspaces")

	checker := &mockSessionChecker{
		activeSessions: make(map[string]bool),
	}

	sm := NewSemaphoreManager(basePath, checker)
	workspaceID := "ws-concurrent-test"

	// Run concurrent operations
	done := make(chan bool)
	errors := make(chan error, 20)

	// Start 10 goroutines acquiring semaphores
	for i := 0; i < 10; i++ {
		go func(id int) {
			holder := Holder{
				ID:        string(rune('a' + id)),
				Type:      HolderTypeSession,
				SessionID: string(rune('a' + id)),
			}
			if err := sm.Acquire(workspaceID, holder); err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Start 10 goroutines reading holders
	for i := 0; i < 10; i++ {
		go func() {
			if _, err := sm.GetHolders(workspaceID); err != nil {
				errors <- err
			}
			done <- true
		}()
	}

	// Wait for all operations to complete
	for i := 0; i < 20; i++ {
		<-done
	}
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation failed: %v", err)
	}

	// Verify final state
	holders, err := sm.GetHolders(workspaceID)
	require.NoError(t, err)
	assert.Len(t, holders, 10)
}

func TestSemaphoreManager_NonExistentWorkspace(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "workspaces")

	sm := NewSemaphoreManager(basePath, nil)

	// Get holders for non-existent workspace
	holders, err := sm.GetHolders("non-existent")
	require.NoError(t, err)
	assert.Empty(t, holders)

	// Check if non-existent workspace is in use
	inUse, _, err := sm.IsInUse("non-existent")
	require.NoError(t, err)
	assert.False(t, inUse)
}

func TestSemaphoreManager_CorruptedFile(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "workspaces")
	workspaceID := "ws-corrupted"

	// Create directory and corrupted semaphore file
	wsDir := filepath.Join(basePath, workspaceID)
	err := os.MkdirAll(wsDir, 0o755)
	require.NoError(t, err)

	semaphorePath := filepath.Join(wsDir, semaphoreFileName)
	err = os.WriteFile(semaphorePath, []byte("invalid json"), 0o644)
	require.NoError(t, err)

	sm := NewSemaphoreManager(basePath, nil)

	// Try to get holders - should fail
	_, err = sm.GetHolders(workspaceID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse semaphore file")
}

func TestHolder_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		holder   Holder
		expected bool
	}{
		{
			name: "session holder not expired",
			holder: Holder{
				Type:      HolderTypeSession,
				Timestamp: time.Now(),
			},
			expected: false,
		},
		{
			name: "cli holder not expired",
			holder: Holder{
				Type:      HolderTypeCLI,
				Timestamp: time.Now().Add(-2 * time.Minute),
			},
			expected: false,
		},
		{
			name: "cli holder expired",
			holder: Holder{
				Type:      HolderTypeCLI,
				Timestamp: time.Now().Add(-10 * time.Minute),
			},
			expected: true,
		},
		{
			name: "unknown type expired",
			holder: Holder{
				Type:      "unknown",
				Timestamp: time.Now(),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.holder.IsExpired())
		})
	}
}
