package common

import (
	"fmt"
	"sync"
)

// ShortIDGenerator generates short sequential IDs
type ShortIDGenerator struct {
	mu      sync.Mutex
	counter int
}

// NewShortIDGenerator creates a new short ID generator
func NewShortIDGenerator() *ShortIDGenerator {
	return &ShortIDGenerator{}
}

// Next returns the next sequential ID
func (g *ShortIDGenerator) Next() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.counter++
	return fmt.Sprintf("%d", g.counter)
}

// SetCounter sets the counter to a specific value (useful for loading state)
func (g *ShortIDGenerator) SetCounter(value int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.counter = value
}

// GetCounter returns the current counter value
func (g *ShortIDGenerator) GetCounter() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.counter

}
