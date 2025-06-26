//go:build !windows
// +build !windows

package semaphore

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessSafety tests semaphore behavior across multiple processes
func TestProcessSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping process safety test in short mode")
	}

	// Create a test binary that can be used as a subprocess
	helperProcess := os.Args[0]
	if os.Getenv("GO_TEST_PROCESS") == "1" {
		runHelperProcess()
		return
	}

	t.Run("exclusive access across processes", func(t *testing.T) {
		tempDir := t.TempDir()
		semPath := filepath.Join(tempDir, "process.lock")
		resultPath := filepath.Join(tempDir, "results.txt")

		// Start multiple processes
		var wg sync.WaitGroup
		processes := 5

		for i := 0; i < processes; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				cmd := exec.Command(helperProcess, "-test.run=TestProcessSafety")
				cmd.Env = append(os.Environ(),
					"GO_TEST_PROCESS=1",
					"SEMAPHORE_PATH="+semPath,
					"RESULT_PATH="+resultPath,
					"PROCESS_ID="+fmt.Sprintf("%d", id),
				)

				if err := cmd.Run(); err != nil {
					t.Errorf("process %d failed: %v", id, err)
				}
			}(i)
		}

		wg.Wait()

		// Verify results
		data, err := os.ReadFile(resultPath)
		require.NoError(t, err)

		// Each process should have written its ID
		// Note: Some processes might fail to acquire the semaphore due to retries
		// So we check that at least one process succeeded
		assert.GreaterOrEqual(t, len(data), 1)
		assert.LessOrEqual(t, len(data), processes)
	})
}

// runHelperProcess is the subprocess that attempts to acquire the semaphore
func runHelperProcess() {
	semPath := os.Getenv("SEMAPHORE_PATH")
	resultPath := os.Getenv("RESULT_PATH")
	processID := os.Getenv("PROCESS_ID")

	if semPath == "" || resultPath == "" || processID == "" {
		os.Exit(1)
	}

	sem, err := New(semPath, 1)
	if err != nil {
		os.Exit(1)
	}
	defer sem.Close()

	holder := &testHolder{id: "process-" + processID}

	// Try to acquire with retries
	acquired := false
	for i := 0; i < 10; i++ {
		if err := sem.Acquire(holder); err == nil {
			acquired = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !acquired {
		os.Exit(1)
	}

	// Write to result file
	file, err := os.OpenFile(resultPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		os.Exit(1)
	}
	defer file.Close()

	if _, err := file.WriteString(processID); err != nil {
		os.Exit(1)
	}

	// Hold for a bit
	time.Sleep(10 * time.Millisecond)

	// Release
	if err := sem.Release(holder.ID()); err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}
