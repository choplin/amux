package common

import (
	"testing"
)

func TestShortIDGenerator(t *testing.T) {
	gen := NewShortIDGenerator()

	// Test sequential generation
	id1 := gen.Next()
	if id1 != "1" {
		t.Errorf("Expected first ID to be '1', got '%s'", id1)
	}

	id2 := gen.Next()
	if id2 != "2" {
		t.Errorf("Expected second ID to be '2', got '%s'", id2)
	}

	id3 := gen.Next()
	if id3 != "3" {
		t.Errorf("Expected third ID to be '3', got '%s'", id3)
	}

	// Test counter getter
	if gen.GetCounter() != 3 {
		t.Errorf("Expected counter to be 3, got %d", gen.GetCounter())
	}

	// Test counter setter
	gen.SetCounter(10)
	id11 := gen.Next()
	if id11 != "11" {
		t.Errorf("Expected ID after setting counter to 10 to be '11', got '%s'", id11)
	}
}

func TestShortIDGeneratorConcurrency(t *testing.T) {
	gen := NewShortIDGenerator()

	// Test concurrent access
	done := make(chan bool)
	ids := make(map[string]bool)

	// Generate IDs concurrently
	for i := 0; i < 100; i++ {
		go func() {
			id := gen.Next()
			ids[id] = true
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Check we got 100 unique IDs
	if len(ids) != 100 {
		t.Errorf("Expected 100 unique IDs, got %d", len(ids))
	}

}
