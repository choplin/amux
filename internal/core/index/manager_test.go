package index

import (
	"fmt"
	"sort"
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

func TestManager_Reconcile(t *testing.T) {
	tempDir := t.TempDir()
	manager, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create some workspace indices
	idx1, err := manager.Acquire(EntityTypeWorkspace, "ws1")
	if err != nil {
		t.Fatalf("Failed to acquire index for ws1: %v", err)
	}
	idx2, err := manager.Acquire(EntityTypeWorkspace, "ws2")
	if err != nil {
		t.Fatalf("Failed to acquire index for ws2: %v", err)
	}
	idx3, err := manager.Acquire(EntityTypeWorkspace, "ws3")
	if err != nil {
		t.Fatalf("Failed to acquire index for ws3: %v", err)
	}

	// Verify all indices are active
	if _, exists := manager.GetByIndex(EntityTypeWorkspace, idx1); !exists {
		t.Error("Expected ws1 index to exist")
	}
	if _, exists := manager.GetByIndex(EntityTypeWorkspace, idx2); !exists {
		t.Error("Expected ws2 index to exist")
	}
	if _, exists := manager.GetByIndex(EntityTypeWorkspace, idx3); !exists {
		t.Error("Expected ws3 index to exist")
	}

	// Reconcile with only ws2 existing
	existingIDs := []string{"ws2"}
	orphanedCount, err := manager.Reconcile(EntityTypeWorkspace, existingIDs)
	if err != nil {
		t.Fatalf("Failed to reconcile: %v", err)
	}

	// Should have cleaned up 2 entries (ws1 and ws3)
	if orphanedCount != 2 {
		t.Errorf("Expected 2 orphaned entries, got %d", orphanedCount)
	}

	// Verify ws1 and ws3 are no longer active
	if _, exists := manager.GetByIndex(EntityTypeWorkspace, idx1); exists {
		t.Error("Expected ws1 index to be removed")
	}
	if _, exists := manager.GetByIndex(EntityTypeWorkspace, idx3); exists {
		t.Error("Expected ws3 index to be removed")
	}

	// Verify ws2 is still active
	wsID, exists := manager.GetByIndex(EntityTypeWorkspace, idx2)
	if !exists {
		t.Error("Expected ws2 index to still exist")
	}
	if wsID != "ws2" {
		t.Errorf("Expected ws2, got %s", wsID)
	}

	// Verify released indices can be reused
	idx4, err := manager.Acquire(EntityTypeWorkspace, "ws4")
	if err != nil {
		t.Fatalf("Failed to acquire index for ws4: %v", err)
	}

	// Should reuse one of the released indices (1 or 3)
	if idx4 != idx1 && idx4 != idx3 {
		t.Errorf("Expected reused index to be %d or %d, got %d", idx1, idx3, idx4)
	}
}

func TestManager_ReconcileEmpty(t *testing.T) {
	tempDir := t.TempDir()
	manager, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create some indices
	manager.Acquire(EntityTypeSession, "sess1")
	manager.Acquire(EntityTypeSession, "sess2")

	// Reconcile with empty list (all should be removed)
	orphanedCount, err := manager.Reconcile(EntityTypeSession, []string{})
	if err != nil {
		t.Fatalf("Failed to reconcile: %v", err)
	}

	if orphanedCount != 2 {
		t.Errorf("Expected 2 orphaned entries, got %d", orphanedCount)
	}

	// Verify all are removed
	if _, exists := manager.GetByIndex(EntityTypeSession, 1); exists {
		t.Error("Expected session 1 to be removed")
	}
	if _, exists := manager.GetByIndex(EntityTypeSession, 2); exists {
		t.Error("Expected session 2 to be removed")
	}
}

func TestManager_ReconcileNoop(t *testing.T) {
	tempDir := t.TempDir()
	manager, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create indices
	manager.Acquire(EntityTypeWorkspace, "ws1")
	manager.Acquire(EntityTypeWorkspace, "ws2")

	// Reconcile with all existing (no changes)
	existingIDs := []string{"ws1", "ws2"}
	orphanedCount, err := manager.Reconcile(EntityTypeWorkspace, existingIDs)
	if err != nil {
		t.Fatalf("Failed to reconcile: %v", err)
	}

	if orphanedCount != 0 {
		t.Errorf("Expected 0 orphaned entries, got %d", orphanedCount)
	}

	// Verify all still exist
	if _, exists := manager.Get(EntityTypeWorkspace, "ws1"); !exists {
		t.Error("Expected ws1 to still exist")
	}
	if _, exists := manager.Get(EntityTypeWorkspace, "ws2"); !exists {
		t.Error("Expected ws2 to still exist")
	}
}

func TestManager_ReconcileReleasedOrder(t *testing.T) {
	tempDir := t.TempDir()
	manager, err := NewManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create multiple indices
	indices := make([]int, 5)
	for i := 0; i < 5; i++ {
		idx, _ := manager.Acquire(EntityTypeWorkspace, fmt.Sprintf("ws%d", i+1))
		indices[i] = int(idx)
	}

	// Reconcile to remove ws1, ws3, ws5 (indices 1, 3, 5)
	existingIDs := []string{"ws2", "ws4"}
	manager.Reconcile(EntityTypeWorkspace, existingIDs)

	// Create new workspaces and verify they get the lowest available indices
	newIdx1, _ := manager.Acquire(EntityTypeWorkspace, "new1")
	newIdx2, _ := manager.Acquire(EntityTypeWorkspace, "new2")
	newIdx3, _ := manager.Acquire(EntityTypeWorkspace, "new3")

	// Collect the new indices
	newIndices := []int{int(newIdx1), int(newIdx2), int(newIdx3)}
	sort.Ints(newIndices)

	// Should have gotten indices 1, 3, 5 (in sorted order)
	expectedIndices := []int{1, 3, 5}
	for i, expected := range expectedIndices {
		if newIndices[i] != expected {
			t.Errorf("Expected index %d at position %d, got %d", expected, i, newIndices[i])
		}
	}
}
