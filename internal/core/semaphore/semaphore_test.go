package semaphore

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHolder implements the Holder interface for testing.
type mockHolder struct {
	id string
}

func (m mockHolder) ID() string {
	return m.id
}

func TestFileSemaphore_AcquireRelease(t *testing.T) {
	tmpDir := t.TempDir()
	semPath := filepath.Join(tmpDir, "test.sem")

	sem := New(semPath, 2)

	holder1 := mockHolder{id: "holder-1"}
	holder2 := mockHolder{id: "holder-2"}
	holder3 := mockHolder{id: "holder-3"}

	// Test acquire
	err := sem.Acquire(holder1)
	require.NoError(t, err)
	assert.Equal(t, 1, sem.Count())
	assert.Equal(t, 1, sem.Available())

	// Test double acquire (should fail)
	err = sem.Acquire(holder1)
	assert.Error(t, err)
	var alreadyErr ErrAlreadyHolder
	assert.ErrorAs(t, err, &alreadyErr)

	// Acquire second holder
	err = sem.Acquire(holder2)
	require.NoError(t, err)
	assert.Equal(t, 2, sem.Count())
	assert.Equal(t, 0, sem.Available())

	// Try to acquire when full (should fail)
	err = sem.Acquire(holder3)
	assert.Error(t, err)
	var fullErr ErrSemaphoreFull
	assert.ErrorAs(t, err, &fullErr)

	// Release first holder
	err = sem.Release(holder1)
	require.NoError(t, err)
	assert.Equal(t, 1, sem.Count())
	assert.Equal(t, 1, sem.Available())

	// Now third holder can acquire
	err = sem.Acquire(holder3)
	require.NoError(t, err)
	assert.Equal(t, 2, sem.Count())

	// Test release non-holder (should fail)
	err = sem.Release(holder1)
	assert.Error(t, err)
	var notHolderErr ErrNotHolder
	assert.ErrorAs(t, err, &notHolderErr)
}

func TestFileSemaphore_IsHeld(t *testing.T) {
	tmpDir := t.TempDir()
	semPath := filepath.Join(tmpDir, "test.sem")

	sem := New(semPath, 1)
	holder := mockHolder{id: "test-holder"}

	// Initially not held
	assert.False(t, sem.IsHeld(holder))

	// Acquire and check
	err := sem.Acquire(holder)
	require.NoError(t, err)
	assert.True(t, sem.IsHeld(holder))

	// Release and check
	err = sem.Release(holder)
	require.NoError(t, err)
	assert.False(t, sem.IsHeld(holder))
}

func TestFileSemaphore_Holders(t *testing.T) {
	tmpDir := t.TempDir()
	semPath := filepath.Join(tmpDir, "test.sem")

	sem := New(semPath, 3)

	// Initially empty
	holders := sem.Holders()
	assert.Empty(t, holders)

	// Add holders
	holder1 := mockHolder{id: "holder-1"}
	holder2 := mockHolder{id: "holder-2"}

	err := sem.Acquire(holder1)
	require.NoError(t, err)
	err = sem.Acquire(holder2)
	require.NoError(t, err)

	// Check holders
	holders = sem.Holders()
	assert.Len(t, holders, 2)
	assert.Contains(t, holders, "holder-1")
	assert.Contains(t, holders, "holder-2")
}

func TestFileSemaphore_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	semPath := filepath.Join(tmpDir, "test.sem")

	sem := New(semPath, 2)

	// Add some holders
	err := sem.Acquire(mockHolder{id: "holder-1"})
	require.NoError(t, err)
	err = sem.Acquire(mockHolder{id: "holder-2"})
	require.NoError(t, err)

	assert.Equal(t, 2, sem.Count())

	// Clear
	err = sem.Clear()
	require.NoError(t, err)

	assert.Equal(t, 0, sem.Count())
	assert.Empty(t, sem.Holders())
}

func TestFileSemaphore_Remove(t *testing.T) {
	tmpDir := t.TempDir()
	semPath := filepath.Join(tmpDir, "test.sem")

	sem := New(semPath, 3)

	// Add holders
	holders := []mockHolder{
		{id: "holder-1"},
		{id: "holder-2"},
		{id: "holder-3"},
	}

	for _, h := range holders {
		err := sem.Acquire(h)
		require.NoError(t, err)
	}

	// Remove specific holders
	err := sem.Remove("holder-1", "holder-3")
	require.NoError(t, err)

	remaining := sem.Holders()
	assert.Len(t, remaining, 1)
	assert.Equal(t, "holder-2", remaining[0])

	// Remove non-existent holder (should be idempotent)
	err = sem.Remove("holder-999")
	require.NoError(t, err)
}

func TestFileSemaphore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	semPath := filepath.Join(tmpDir, "test.sem")

	// Create semaphore and add holders
	sem1 := New(semPath, 2)
	holder1 := mockHolder{id: "persistent-1"}
	holder2 := mockHolder{id: "persistent-2"}

	err := sem1.Acquire(holder1)
	require.NoError(t, err)
	err = sem1.Acquire(holder2)
	require.NoError(t, err)

	// Create new semaphore instance pointing to same file
	sem2 := New(semPath, 2)

	// Should see the same holders
	assert.Equal(t, 2, sem2.Count())
	holders := sem2.Holders()
	assert.Contains(t, holders, "persistent-1")
	assert.Contains(t, holders, "persistent-2")

	// Should not be able to acquire (full)
	err = sem2.Acquire(mockHolder{id: "new-holder"})
	assert.Error(t, err)
}

func TestFileSemaphore_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	semPath := filepath.Join(tmpDir, "test.sem")

	sem := New(semPath, 5)

	// Concurrent acquires
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			holder := mockHolder{id: fmt.Sprintf("concurrent-%d", id)}
			if err := sem.Acquire(holder); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Should have exactly 5 errors (capacity exceeded)
	errCount := 0
	for err := range errors {
		errCount++
		var fullErr ErrSemaphoreFull
		assert.ErrorAs(t, err, &fullErr)
	}
	assert.Equal(t, 5, errCount)
	assert.Equal(t, 5, sem.Count())
}

func TestFileSemaphore_CapacityHandling(t *testing.T) {
	tmpDir := t.TempDir()
	semPath := filepath.Join(tmpDir, "test.sem")

	// Test with capacity 0 (should default to 1)
	sem := New(semPath, 0)
	assert.Equal(t, 1, sem.Capacity())

	// Test negative capacity (should default to 1)
	sem = New(semPath, -5)
	assert.Equal(t, 1, sem.Capacity())
}

func TestFileSemaphore_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	semPath := filepath.Join(tmpDir, "test.sem")

	// Write corrupted data
	err := os.WriteFile(semPath, []byte("not json"), 0o644)
	require.NoError(t, err)

	sem := New(semPath, 1)
	holder := mockHolder{id: "test"}

	// Should fail to acquire due to corrupted file
	err = sem.Acquire(holder)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}
