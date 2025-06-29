// Package id provides ID generation functionality
package id

import (
	"fmt"
	"sync/atomic"
)

// Generator generates unique IDs
type Generator interface {
	Generate() string
}

// sequentialGenerator generates sequential numeric IDs
type sequentialGenerator struct {
	prefix  string
	counter uint64
}

// NewSequentialGenerator creates a new sequential ID generator
func NewSequentialGenerator(prefix string) Generator {
	return &sequentialGenerator{
		prefix: prefix,
	}
}

// Generate returns the next ID in sequence
func (g *sequentialGenerator) Generate() string {
	count := atomic.AddUint64(&g.counter, 1)
	if g.prefix != "" {
		return fmt.Sprintf("%s-%d", g.prefix, count)
	}
	return fmt.Sprintf("%d", count)
}
