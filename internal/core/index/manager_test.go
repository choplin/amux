package index

import (
	"fmt"
	"testing"
)

func TestManager_AcquireAndRelease(t *testing.T) {
	tempDir := t.TempDir()
	manager, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test acquiring new indices
	idx1, err := manager.Acquire(EntityTypeWorkspace, "ws1")
	if err != nil {
		t.Fatalf("Failed to acquire index: %v", err)
	}
	if idx1 != 1 {
		t.Errorf("Expected first index to be 1, got %d", idx1)
	}

	idx2, err := manager.Acquire(EntityTypeWorkspace, "ws2")
	if err != nil {
		t.Fatalf("Failed to acquire index: %v", err)
	}
	if idx2 != 2 {
		t.Errorf("Expected second index to be 2, got %d", idx2)
	}

	// Test getting existing index
	idx1Again, err := manager.Acquire(EntityTypeWorkspace, "ws1")
	if err != nil {
		t.Fatalf("Failed to acquire existing index: %v", err)
	}
	if idx1Again != idx1 {
		t.Errorf("Expected same index for existing entity, got %d", idx1Again)
	}

	// Test release and reuse
	err = manager.Release(EntityTypeWorkspace, "ws1")
	if err != nil {
		t.Fatalf("Failed to release index: %v", err)
	}

	idx3, err := manager.Acquire(EntityTypeWorkspace, "ws3")
	if err != nil {
		t.Fatalf("Failed to acquire index after release: %v", err)
	}
	if idx3 != 1 {
		t.Errorf("Expected reused index 1, got %d", idx3)
	}
}

func TestManager_Get(t *testing.T) {
	tempDir := t.TempDir()
	manager, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Acquire some indices
	manager.Acquire(EntityTypeWorkspace, "ws1")
	manager.Acquire(EntityTypeWorkspace, "ws2")
	manager.Acquire(EntityTypeSession, "sess1")

	// Test Get
	idx, found := manager.Get(EntityTypeWorkspace, "ws1")
	if !found {
		t.Error("Expected to find index for ws1")
	}
	if idx != 1 {
		t.Errorf("Expected index 1, got %d", idx)
	}

	_, found = manager.Get(EntityTypeWorkspace, "ws3")
	if found {
		t.Error("Expected not to find index for ws3")
	}

	// Test GetByIndex
	entityID, found := manager.GetByIndex(EntityTypeWorkspace, 2)
	if !found {
		t.Error("Expected to find entity for index 2")
	}
	if entityID != "ws2" {
		t.Errorf("Expected entity ws2, got %s", entityID)
	}
}

func TestManager_Persistence(t *testing.T) {
	tempDir := t.TempDir()

	// Create manager and acquire some indices
	manager1, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	manager1.Acquire(EntityTypeWorkspace, "ws1")
	manager1.Acquire(EntityTypeWorkspace, "ws2")
	manager1.Release(EntityTypeWorkspace, "ws1")

	// Create new manager instance (should load state)
	manager2, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create second manager: %v", err)
	}

	// Verify state was persisted
	idx, found := manager2.Get(EntityTypeWorkspace, "ws2")
	if !found {
		t.Error("Expected to find ws2 in loaded state")
	}
	if idx != 2 {
		t.Errorf("Expected index 2, got %d", idx)
	}

	// Verify released index is reused
	idx3, err := manager2.Acquire(EntityTypeWorkspace, "ws3")
	if err != nil {
		t.Fatalf("Failed to acquire index: %v", err)
	}
	if idx3 != 1 {
		t.Errorf("Expected reused index 1, got %d", idx3)
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	manager, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test concurrent acquisitions
	done := make(chan bool)
	indices := make(chan Index, 100)

	for i := 0; i < 100; i++ {
		go func(i int) {
			idx, err := manager.Acquire(EntityTypeWorkspace, fmt.Sprintf("ws%d", i))
			if err != nil {
				t.Errorf("Failed to acquire index: %v", err)
			}
			indices <- idx
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
	close(indices)

	// Verify all indices are unique
	seen := make(map[Index]bool)
	for idx := range indices {
		if seen[idx] {
			t.Errorf("Duplicate index: %d", idx)
		}
		seen[idx] = true
	}

	if len(seen) != 100 {
		t.Errorf("Expected 100 unique indices, got %d", len(seen))
	}
}

func TestManager_MultipleEntityTypes(t *testing.T) {
	tempDir := t.TempDir()
	manager, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Acquire indices for different entity types
	wsIdx, _ := manager.Acquire(EntityTypeWorkspace, "ws1")
	sessIdx, _ := manager.Acquire(EntityTypeSession, "sess1")

	// Both should be 1 since they're separate counters
	if wsIdx != 1 {
		t.Errorf("Expected workspace index 1, got %d", wsIdx)
	}
	if sessIdx != 1 {
		t.Errorf("Expected session index 1, got %d", sessIdx)
	}

	// Verify they're tracked separately
	wsID, _ := manager.GetByIndex(EntityTypeWorkspace, 1)
	sessID, _ := manager.GetByIndex(EntityTypeSession, 1)

	if wsID != "ws1" {
		t.Errorf("Expected ws1, got %s", wsID)
	}
	if sessID != "sess1" {
		t.Errorf("Expected sess1, got %s", sessID)
	}

}
