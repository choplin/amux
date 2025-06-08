package workspace

import "time"

// Status represents the current status of a workspace
type Status string

const (
	StatusActive Status = "active"
	StatusIdle   Status = "idle"
)

// Workspace represents an isolated development environment
type Workspace struct {
	ID          string    `yaml:"id" json:"id"`
	Name        string    `yaml:"name" json:"name"`
	Status      Status    `yaml:"status" json:"status"`
	Branch      string    `yaml:"branch" json:"branch"`
	BaseBranch  string    `yaml:"baseBranch" json:"baseBranch"`
	Path        string    `yaml:"path" json:"path"`
	AgentID     string    `yaml:"agentId,omitempty" json:"agentId,omitempty"`
	Description string    `yaml:"description,omitempty" json:"description,omitempty"`
	CreatedAt   time.Time `yaml:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time `yaml:"updatedAt" json:"updatedAt"`
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
	Status Status
}

// CleanupOptions represents options for cleaning up old workspaces
type CleanupOptions struct {
	Days   int
	DryRun bool
}
