package index

import (
	"fmt"
)

// Index represents a numeric index for convenient reference
type Index int

// String returns the string representation of the index
func (i Index) String() string {
	return fmt.Sprintf("%d", i)
}

// IsValid checks if the index is valid (positive)
func (i Index) IsValid() bool {
	return i > 0
}

// EntityType represents the type of entity being indexed
type EntityType string

const (
	// EntityTypeWorkspace represents a workspace entity
	EntityTypeWorkspace EntityType = "workspace"
	// EntityTypeSession represents a session entity
	EntityTypeSession EntityType = "session"
)

// State represents the internal state of the index manager
type State struct {
	// Counters track the highest index allocated for each entity type
	Counters map[EntityType]int `yaml:"counters"`

	// Active tracks currently active index assignments (index -> entityID)
	Active map[EntityType]map[int]string `yaml:"active"`

	// Released tracks indices available for reuse
	Released map[EntityType][]int `yaml:"released"`
}

// NewState creates a new empty state
func NewState() *State {
	return &State{
		Counters: make(map[EntityType]int),
		Active:   make(map[EntityType]map[int]string),
		Released: make(map[EntityType][]int),
	}
}
