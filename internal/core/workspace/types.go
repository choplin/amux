package workspace

import "time"

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
	CreatedAt   time.Time `yaml:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time `yaml:"-" json:"updatedAt"` // Dynamically populated from filesystem
}

// CreateOptions represents options for creating a new workspace
type CreateOptions struct {
	Name        string
	BaseBranch  string
	Branch      string // Specify existing branch to use
	AgentID     string
	Description string
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
