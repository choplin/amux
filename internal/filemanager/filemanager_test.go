package filemanager

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type TestData struct {
	Name    string `yaml:"name"`
	Value   int    `yaml:"value"`
	Updated bool   `yaml:"updated"`
}

func TestManager_ReadWrite(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")

	mgr := NewManager[TestData]()

	// Test write
	data := &TestData{
		Name:  "test",
		Value: 42,
	}

	err := mgr.Write(context.Background(), testFile, data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Test read
	readData, info, err := mgr.Read(context.Background(), testFile)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if readData.Name != data.Name || readData.Value != data.Value {
		t.Errorf("Read data mismatch: got %+v, want %+v", readData, data)
	}

	if info.Path != testFile {
		t.Errorf("FileInfo path mismatch: got %s, want %s", info.Path, testFile)
	}
}

func TestManager_CAS(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")

	mgr := NewManager[TestData]()

	// Initial write
	data := &TestData{
		Name:  "test",
		Value: 1,
	}
	err := mgr.Write(context.Background(), testFile, data)
	if err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	// Read to get file info
	_, info, err := mgr.Read(context.Background(), testFile)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// CAS write should succeed
	data.Value = 2
	err = mgr.WriteWithCAS(context.Background(), testFile, data, info)
	if err != nil {
		t.Fatalf("CAS write failed: %v", err)
	}

	// Simulate concurrent modification by touching the file
	time.Sleep(10 * time.Millisecond) // Ensure different timestamp
	os.Chtimes(testFile, time.Now(), time.Now())

	// CAS write should fail
	data.Value = 3
	err = mgr.WriteWithCAS(context.Background(), testFile, data, info)
	if !errors.Is(err, ErrConcurrentModification) {
		t.Errorf("Expected ErrConcurrentModification, got: %v", err)
	}
}

func TestManager_Update(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")

	mgr := NewManager[TestData]()

	// Test update on non-existent file
	err := mgr.Update(context.Background(), testFile, func(data *TestData) error {
		data.Name = "created"
		data.Value = 100
		return nil
	})
	if err != nil {
		t.Fatalf("Update on new file failed: %v", err)
	}

	// Verify file was created
	readData, _, err := mgr.Read(context.Background(), testFile)
	if err != nil {
		t.Fatalf("Read after create failed: %v", err)
	}
	if readData.Name != "created" || readData.Value != 100 {
		t.Errorf("Created data mismatch: got %+v", readData)
	}

	// Test update on existing file
	err = mgr.Update(context.Background(), testFile, func(data *TestData) error {
		data.Value = 200
		data.Updated = true
		return nil
	})
	if err != nil {
		t.Fatalf("Update on existing file failed: %v", err)
	}

	// Verify update
	readData, _, err = mgr.Read(context.Background(), testFile)
	if err != nil {
		t.Fatalf("Read after update failed: %v", err)
	}
	if readData.Value != 200 || !readData.Updated {
		t.Errorf("Updated data mismatch: got %+v", readData)
	}
}

func TestManager_ConcurrentUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")

	mgr := NewManager[TestData]()

	// Initial data
	err := mgr.Write(context.Background(), testFile, &TestData{Value: 0})
	if err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	// Run concurrent updates
	const numGoroutines = 10
	const incrementsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				err := mgr.Update(context.Background(), testFile, func(data *TestData) error {
					data.Value++
					return nil
				})
				if err != nil {
					t.Errorf("Update failed: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	// Verify final value
	readData, _, err := mgr.Read(context.Background(), testFile)
	if err != nil {
		t.Fatalf("Final read failed: %v", err)
	}

	expectedValue := numGoroutines * incrementsPerGoroutine
	if readData.Value != expectedValue {
		t.Errorf("Final value mismatch: got %d, want %d", readData.Value, expectedValue)
	}
}

func TestManager_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")

	mgr := NewManager[TestData]()

	// Create file
	err := mgr.Write(context.Background(), testFile, &TestData{Name: "test"})
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Delete file
	err = mgr.Delete(context.Background(), testFile)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify file is gone
	_, _, err = mgr.Read(context.Background(), testFile)
	if err == nil || !os.IsNotExist(err) {
		t.Errorf("Expected file not exist error, got: %v", err)
	}

	// Delete non-existent file should not error
	err = mgr.Delete(context.Background(), testFile)
	if err != nil {
		t.Errorf("Delete non-existent file failed: %v", err)
	}
}

func TestManager_Timeout(t *testing.T) {
	// Skip this test in short mode as it involves timeouts
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")

	// Create manager with very short timeout
	mgr := NewManagerWithTimeout[TestData](10 * time.Millisecond)

	// Create initial file
	err := mgr.Write(context.Background(), testFile, &TestData{Name: "test"})
	if err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	// This test would require simulating a locked file, which is platform-specific
	// For now, we just verify the manager was created with custom timeout
	if mgr.lockTimeout != 10*time.Millisecond {
		t.Errorf("Lock timeout not set correctly: got %v, want %v", mgr.lockTimeout, 10*time.Millisecond)
	}
}

func TestManager_UpdateError(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.yaml")

	mgr := NewManager[TestData]()

	// Create initial file
	err := mgr.Write(context.Background(), testFile, &TestData{Name: "test"})
	if err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	// Test update function that returns error
	testErr := errors.New("update error")
	err = mgr.Update(context.Background(), testFile, func(data *TestData) error {
		return testErr
	})

	if err == nil {
		t.Fatal("Expected error from update function")
	}

	if !errors.Is(err, testErr) {
		t.Errorf("Expected wrapped update error, got: %v", err)
	}
}
