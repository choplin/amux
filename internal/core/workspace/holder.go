package workspace

import (
	"time"
)

// HolderType represents the type of entity holding a semaphore
type HolderType string

const (
	// HolderTypeSession represents a session holding the semaphore
	HolderTypeSession HolderType = "session"
	// HolderTypeCLI represents a CLI command holding the semaphore
	HolderTypeCLI HolderType = "cli"
)

// Holder represents an entity holding a workspace semaphore
type Holder struct {
	ID          string     `json:"id"`           // Unique holder ID (usually session ID)
	Type        HolderType `json:"type"`         // Type of holder
	SessionID   string     `json:"session_id"`   // Associated session ID (if applicable)
	WorkspaceID string     `json:"workspace_id"` // Workspace being held
	Timestamp   time.Time  `json:"timestamp"`    // When the semaphore was acquired
	Description string     `json:"description"`  // Human-readable description
}

// IsExpired checks if a holder has expired based on its type
func (h *Holder) IsExpired() bool {
	switch h.Type {
	case HolderTypeCLI:
		// CLI commands have a 5-minute timeout
		return time.Since(h.Timestamp) > 5*time.Minute
	case HolderTypeSession:
		// Sessions don't expire based on time alone
		return false
	default:
		// Unknown types are considered expired
		return true
	}
}
