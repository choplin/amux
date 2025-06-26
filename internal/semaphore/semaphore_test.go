package semaphore

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testHolder is a simple holder implementation for testing
type testHolder struct {
	id string
}

func (h *testHolder) ID() string {
	return h.id
}

func TestNew(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		capacity         int
		expectedCapacity int
	}{
		{
			name:             "default capacity",
			path:             "/tmp/test.lock",
			capacity:         0,
			expectedCapacity: 1,
		},
		{
			name:             "custom capacity",
			path:             "/tmp/test.lock",
			capacity:         5,
			expectedCapacity: 5,
		},
		{
			name:             "negative capacity",
			path:             "/tmp/test.lock",
			capacity:         -1,
			expectedCapacity: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			path := filepath.Join(tempDir, tt.path)
			sem, err := New(path, tt.capacity)
			require.NoError(t, err)
			defer sem.Close()

			assert.Equal(t, path, sem.path)
			assert.Equal(t, tt.expectedCapacity, sem.capacity)
		})
	}
}

func TestAcquire(t *testing.T) {
	t.Run("single holder", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")
		sem, err := New(semPath, 1)
		require.NoError(t, err)
		defer sem.Close()

		holder := &testHolder{id: "holder1"}
		err = sem.Acquire(holder)
		require.NoError(t, err)

		// Verify holder is recorded
		holders := sem.Holders()
		assert.Equal(t, []string{"holder1"}, holders)
		assert.Equal(t, 1, sem.Count())
		assert.Equal(t, 0, sem.Available())
	})

	t.Run("multiple holders with capacity", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")
		sem, err := New(semPath, 3)
		require.NoError(t, err)
		defer sem.Close()

		holders := []*testHolder{
			{id: "holder1"},
			{id: "holder2"},
			{id: "holder3"},
		}

		for _, h := range holders {
			err := sem.Acquire(h)
			require.NoError(t, err)
		}

		assert.Equal(t, 3, sem.Count())
		assert.Equal(t, 0, sem.Available())
	})

	t.Run("already held", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")
		sem, err := New(semPath, 2)
		require.NoError(t, err)
		defer sem.Close()

		holder := &testHolder{id: "holder1"}

		err = sem.Acquire(holder)
		require.NoError(t, err)

		// Try to acquire again
		err = sem.Acquire(holder)
		assert.Equal(t, ErrAlreadyHeld, err)
	})

	t.Run("no capacity", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")
		sem, err := New(semPath, 1)
		require.NoError(t, err)
		defer sem.Close()

		holder1 := &testHolder{id: "holder1"}
		holder2 := &testHolder{id: "holder2"}

		err = sem.Acquire(holder1)
		require.NoError(t, err)

		err = sem.Acquire(holder2)
		assert.Equal(t, ErrNoCapacity, err)
	})
}

func TestRelease(t *testing.T) {
	t.Run("release held semaphore", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")
		sem, err := New(semPath, 1)
		require.NoError(t, err)
		defer sem.Close()

		holder := &testHolder{id: "holder1"}

		err = sem.Acquire(holder)
		require.NoError(t, err)

		err = sem.Release("holder1")
		require.NoError(t, err)

		assert.Equal(t, 0, sem.Count())
		assert.Equal(t, 1, sem.Available())
	})

	t.Run("release not held", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")
		sem, err := New(semPath, 1)
		require.NoError(t, err)
		defer sem.Close()

		err = sem.Release("holder1")
		assert.Equal(t, ErrNotHeld, err)
	})

	t.Run("release one of multiple", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")
		sem, err := New(semPath, 3)
		require.NoError(t, err)
		defer sem.Close()

		holders := []*testHolder{
			{id: "holder1"},
			{id: "holder2"},
			{id: "holder3"},
		}

		for _, h := range holders {
			err := sem.Acquire(h)
			require.NoError(t, err)
		}

		err = sem.Release("holder2")
		require.NoError(t, err)

		holderIDs := sem.Holders()
		assert.Contains(t, holderIDs, "holder1")
		assert.NotContains(t, holderIDs, "holder2")
		assert.Contains(t, holderIDs, "holder3")
		assert.Equal(t, 2, sem.Count())
	})
}

func TestRemove(t *testing.T) {
	t.Run("remove single holder", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")
		sem, err := New(semPath, 3)
		require.NoError(t, err)
		defer sem.Close()

		holders := []*testHolder{
			{id: "holder1"},
			{id: "holder2"},
			{id: "holder3"},
		}

		for _, h := range holders {
			err := sem.Acquire(h)
			require.NoError(t, err)
		}

		err = sem.Remove("holder2")
		require.NoError(t, err)

		assert.Equal(t, 2, sem.Count())
		holderIDs := sem.Holders()
		assert.NotContains(t, holderIDs, "holder2")
	})

	t.Run("remove multiple holders", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")
		sem, err := New(semPath, 3)
		require.NoError(t, err)
		defer sem.Close()

		holders := []*testHolder{
			{id: "holder1"},
			{id: "holder2"},
			{id: "holder3"},
		}

		for _, h := range holders {
			err := sem.Acquire(h)
			require.NoError(t, err)
		}

		err = sem.Remove("holder1", "holder3")
		require.NoError(t, err)

		assert.Equal(t, 1, sem.Count())
		assert.Equal(t, []string{"holder2"}, sem.Holders())
	})

	t.Run("remove non-existent", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")
		sem, err := New(semPath, 1)
		require.NoError(t, err)
		defer sem.Close()

		holder := &testHolder{id: "holder1"}

		err = sem.Acquire(holder)
		require.NoError(t, err)

		// Removing non-existent should not error
		err = sem.Remove("holder2", "holder3")
		require.NoError(t, err)

		assert.Equal(t, 1, sem.Count())
	})

	t.Run("remove empty list", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")
		sem, err := New(semPath, 1)
		require.NoError(t, err)
		defer sem.Close()

		err = sem.Remove()
		require.NoError(t, err)
	})
}

func TestPersistence(t *testing.T) {
	t.Run("persist and reload", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")

		// Create and acquire
		sem1, err := New(semPath, 2)
		require.NoError(t, err)

		holder1 := &testHolder{id: "holder1"}
		holder2 := &testHolder{id: "holder2"}

		err = sem1.Acquire(holder1)
		require.NoError(t, err)
		err = sem1.Acquire(holder2)
		require.NoError(t, err)

		// Close first instance
		sem1.Close()

		// Create new instance and verify state
		sem2, err := New(semPath, 2)
		require.NoError(t, err)
		defer sem2.Close()

		holders := sem2.Holders()
		assert.Equal(t, 2, len(holders))
		assert.Contains(t, holders, "holder1")
		assert.Contains(t, holders, "holder2")
		assert.Equal(t, 0, sem2.Available())
	})

	t.Run("handle missing file", func(t *testing.T) {
		tempDir := t.TempDir()
		missingPath := filepath.Join(tempDir, "missing.lock")
		sem, err := New(missingPath, 1)
		require.NoError(t, err)
		defer sem.Close()

		// Should work with missing file
		assert.Equal(t, 0, sem.Count())
		assert.Equal(t, 1, sem.Available())
		assert.Empty(t, sem.Holders())

		// Should create file on acquire
		holder := &testHolder{id: "holder1"}
		err = sem.Acquire(holder)
		require.NoError(t, err)

		// File should exist now
		_, err = os.Stat(missingPath)
		require.NoError(t, err)
	})
}

func TestConcurrentAccess(t *testing.T) {
	t.Run("concurrent acquire", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")
		sem, err := New(semPath, 5)
		require.NoError(t, err)
		defer sem.Close()

		var wg sync.WaitGroup
		errors := make(chan error, 10)

		// Start 10 goroutines trying to acquire
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				holder := &testHolder{id: fmt.Sprintf("holder%d", id)}
				err := sem.Acquire(holder)
				if err != nil && err != ErrNoCapacity {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for unexpected errors
		for err := range errors {
			t.Errorf("unexpected error: %v", err)
		}

		// Should have exactly 5 holders
		assert.Equal(t, 5, sem.Count())
		assert.Equal(t, 0, sem.Available())
	})

	t.Run("concurrent acquire and release", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")
		sem, err := New(semPath, 3)
		require.NoError(t, err)
		defer sem.Close()

		var wg sync.WaitGroup
		done := make(chan bool)

		// Continuously acquire and release
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				holder := &testHolder{id: fmt.Sprintf("holder%d", id)}

				for {
					select {
					case <-done:
						return
					default:
						if err := sem.Acquire(holder); err == nil {
							time.Sleep(10 * time.Millisecond)
							sem.Release(holder.ID())
						}
						time.Sleep(5 * time.Millisecond)
					}
				}
			}(i)
		}

		// Let it run for a bit
		time.Sleep(100 * time.Millisecond)
		close(done)
		wg.Wait()

		// Should end with no holders
		assert.Equal(t, 0, sem.Count())
		assert.Equal(t, 3, sem.Available())
	})
}

func TestAtomicOperations(t *testing.T) {
	t.Run("atomic file write", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "test.lock")
		sem, err := New(semPath, 1)
		require.NoError(t, err)
		defer sem.Close()

		holder := &testHolder{id: "holder1"}

		// Acquire to create file
		err = sem.Acquire(holder)
		require.NoError(t, err)

		// Check temp file doesn't exist
		tempFile := semPath + ".tmp"
		_, err = os.Stat(tempFile)
		assert.True(t, os.IsNotExist(err))

		// Release and check again
		err = sem.Release("holder1")
		require.NoError(t, err)

		_, err = os.Stat(tempFile)
		assert.True(t, os.IsNotExist(err))
	})
}
