package index

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// TestManager_ConcurrentProcesses tests that multiple processes can safely
// access the index manager without conflicts
func TestManager_ConcurrentProcesses(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		// We're in a subprocess
		runSubprocess()
		return
	}

	tempDir := t.TempDir()

	// Run multiple processes concurrently
	var wg sync.WaitGroup
	results := make(chan int, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			// Use the test binary itself as the subprocess
			cmd := exec.Command(os.Args[0], "-test.run=TestManager_ConcurrentProcesses")
			cmd.Env = append(os.Environ(),
				"TEST_SUBPROCESS=1",
				fmt.Sprintf("TEST_AMUX_DIR=%s", tempDir),
				fmt.Sprintf("TEST_ENTITY_ID=entity-%d", i),
			)

			output, err := cmd.Output()
			if err != nil {
				t.Errorf("Subprocess failed: %v", err)
				return
			}

			// Extract just the first line (the index)
			lines := strings.Split(string(output), "\n")
			if len(lines) == 0 {
				t.Errorf("No output from subprocess")
				return
			}

			idx, err := strconv.Atoi(lines[0])
			if err != nil {
				t.Errorf("Failed to parse index from '%s': %v", lines[0], err)
				return
			}

			results <- idx
		}(i)
	}

	wg.Wait()
	close(results)

	// Verify all indices are unique and sequential
	seen := make(map[int]bool)
	for idx := range results {
		if seen[idx] {
			t.Errorf("Duplicate index: %d", idx)
		}
		seen[idx] = true
	}

	// Should have 10 unique indices
	if len(seen) != 10 {
		t.Errorf("Expected 10 unique indices, got %d", len(seen))
	}
}

func runSubprocess() {
	amuxDir := os.Getenv("TEST_AMUX_DIR")
	entityID := os.Getenv("TEST_ENTITY_ID")

	manager, err := NewManager(amuxDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create manager: %v\n", err)
		os.Exit(1)
	}

	idx, err := manager.Acquire(EntityTypeWorkspace, entityID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to acquire index: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(idx)
}
