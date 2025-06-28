package workspace

import (
	"time"
)

// ID is the full UUID of a workspace
type ID string

// Index is the short numeric identifier (1, 2, 3...)
type Index string

// Name is the human-readable name
type Name string

// Identifier can be any of: ID, Index, or Name
type Identifier string

// ConsistencyStatus represents the consistency state of a workspace
type ConsistencyStatus int

const (
	// StatusConsistent indicates both folder and git worktree exist
	StatusConsistent ConsistencyStatus = iota
	// StatusFolderMissing indicates git worktree exists but folder is missing
	StatusFolderMissing
	// StatusWorktreeMissing indicates folder exists but git worktree is missing
	StatusWorktreeMissing
	// StatusOrphaned indicates both folder and git worktree are missing
	StatusOrphaned
)

// String returns the string representation of ConsistencyStatus
func (s ConsistencyStatus) String() string {
	switch s {
	case StatusConsistent:
		return "consistent"
	case StatusFolderMissing:
		return "folder-missing"
	case StatusWorktreeMissing:
		return "worktree-missing"
	case StatusOrphaned:
		return "orphaned"
	default:
		return "unknown"
	}
}

// MarshalJSON implements json.Marshaler for ConsistencyStatus
func (s ConsistencyStatus) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler for ConsistencyStatus
func (s *ConsistencyStatus) UnmarshalJSON(data []byte) error {
	str := string(data)
	// Remove quotes
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	switch str {
	case "consistent":
		*s = StatusConsistent
	case "folder-missing":
		*s = StatusFolderMissing
	case "worktree-missing":
		*s = StatusWorktreeMissing
	case "orphaned":
		*s = StatusOrphaned
	default:
		*s = StatusConsistent // Default to consistent if unknown
	}
	return nil
}

// Workspace represents an isolated development environment
type Workspace struct {
	ID          string    `yaml:"id" json:"id"`
	Index       string    `yaml:"-" json:"index"` // Populated from ID mapper, not persisted
	Name        string    `yaml:"name" json:"name"`
	Branch      string    `yaml:"branch" json:"branch"`
	BaseBranch  string    `yaml:"baseBranch" json:"baseBranch"`
	Path        string    `yaml:"path" json:"path"`
	Description string    `yaml:"description,omitempty" json:"description,omitempty"`
	StoragePath string    `yaml:"storagePath,omitempty" json:"storagePath,omitempty"`
	CreatedAt   time.Time `yaml:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time `yaml:"-" json:"updatedAt"` // Dynamically populated from filesystem
	AutoCreated bool      `yaml:"autoCreated,omitempty" json:"autoCreated,omitempty"`

	// Consistency status fields (not persisted)
	PathExists     bool              `yaml:"-" json:"pathExists"`
	WorktreeExists bool              `yaml:"-" json:"worktreeExists"`
	Status         ConsistencyStatus `yaml:"-" json:"status"`
}

// BranchMode specifies how to handle branch creation/checkout
type BranchMode int

const (
	// BranchModeCreate creates a new branch (default)
	BranchModeCreate BranchMode = iota
	// BranchModeCheckout uses an existing branch
	BranchModeCheckout
)

// CreateOptions represents options for creating a new workspace
type CreateOptions struct {
	Name        string
	BaseBranch  string
	Branch      string     // Branch name (either new or existing)
	BranchMode  BranchMode // How to handle the branch (default: BranchModeCreate)
	Description string
	AutoCreated bool // Internal: whether workspace was auto-created by session
	NoHooks     bool // Skip hook execution
}

// ListOptions represents options for listing workspaces
type ListOptions struct {
	// Reserved for future filtering options
}

// RemoveOptions represents options for removing a workspace
type RemoveOptions struct {
	NoHooks         bool   // Skip hook execution
	CurrentDir      string // Current working directory (for safety check)
	SkipSafetyCheck bool   // Skip current directory safety check
}

// CleanupOptions represents options for cleaning up old workspaces
type CleanupOptions struct {
	Days   int
	DryRun bool
}
