package workspace

import "time"

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
	AgentID     string    `yaml:"agentId,omitempty" json:"agentId,omitempty"`
	Description string    `yaml:"description,omitempty" json:"description,omitempty"`
	ContextPath string    `yaml:"contextPath,omitempty" json:"contextPath,omitempty"`
	CreatedAt   time.Time `yaml:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time `yaml:"-" json:"updatedAt"` // Dynamically populated from filesystem
	AutoCreated bool      `yaml:"autoCreated,omitempty" json:"autoCreated,omitempty"`

	// Consistency status fields (not persisted)
	PathExists     bool              `yaml:"-" json:"pathExists"`
	WorktreeExists bool              `yaml:"-" json:"worktreeExists"`
	Status         ConsistencyStatus `yaml:"-" json:"status"`
}

// CreateOptions represents options for creating a new workspace
type CreateOptions struct {
	Name        string
	BaseBranch  string
	Branch      string // Specify existing branch to use
	AgentID     string
	Description string
	AutoCreated bool // Internal: whether workspace was auto-created by session
}

// ListOptions represents options for listing workspaces
type ListOptions struct {
	// Reserved for future filtering options
}

// CleanupOptions represents options for cleaning up old workspaces
type CleanupOptions struct {
	Days   int
	DryRun bool
}
