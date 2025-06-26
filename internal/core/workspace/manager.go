// Package workspace provides management of isolated git worktree-based development environments.
package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/git"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/filemanager"
)

// Manager manages Amux workspaces
type Manager struct {
	configManager *config.Manager
	gitOps        *git.Operations
	workspacesDir string
	idMapper      *idmap.IDMapper
	fm            *filemanager.Manager[Workspace]
}

// NewManager creates a new workspace manager
func NewManager(configManager *config.Manager) (*Manager, error) {
	gitOps := git.NewOperations(configManager.GetProjectRoot())

	if !gitOps.IsGitRepository() {
		return nil, fmt.Errorf("not a git repository")
	}

	workspacesDir := configManager.GetWorkspacesDir()

	// Initialize ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ID mapper: %w", err)
	}

	return &Manager{
		configManager: configManager,
		gitOps:        gitOps,
		workspacesDir: workspacesDir,
		idMapper:      idMapper,
		fm:            filemanager.NewManager[Workspace](),
	}, nil
}

// NewManagerWithIDMapper creates a new workspace manager with a provided ID mapper.
// This allows sharing the ID mapper between multiple managers.
func NewManagerWithIDMapper(configManager *config.Manager, idMapper *idmap.IDMapper) (*Manager, error) {
	gitOps := git.NewOperations(configManager.GetProjectRoot())

	if !gitOps.IsGitRepository() {
		return nil, fmt.Errorf("not a git repository")
	}

	workspacesDir := configManager.GetWorkspacesDir()

	return &Manager{
		configManager: configManager,
		gitOps:        gitOps,
		workspacesDir: workspacesDir,
		idMapper:      idMapper,
		fm:            filemanager.NewManager[Workspace](),
	}, nil
}

// Create creates a new workspace
func (m *Manager) Create(ctx context.Context, opts CreateOptions) (*Workspace, error) {
	// Generate workspace ID
	id := generateID(opts.Name)

	// Determine base branch
	baseBranch := opts.BaseBranch
	if baseBranch == "" {
		var err error
		baseBranch, err = m.gitOps.GetDefaultBranch()
		if err != nil {
			return nil, fmt.Errorf("failed to determine base branch: %w", err)
		}
	}

	// Create workspace directory structure
	workspaceDir := filepath.Join(m.workspacesDir, id)
	worktreePath := filepath.Join(workspaceDir, "worktree")

	// Ensure the workspace directory exists
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Determine branch name
	var branch string
	if opts.Branch != "" {
		branch = opts.Branch
	} else {
		// Auto-generate branch name
		branch = fmt.Sprintf("amux/%s", id)
	}

	// Check if branch exists
	exists, err := m.gitOps.BranchExists(branch)
	if err != nil {
		return nil, fmt.Errorf("failed to check branch existence: %w", err)
	}

	// Handle branch creation/checkout based on mode
	switch opts.BranchMode {
	case BranchModeCreate:
		// Create new branch (fail if exists)
		if exists {
			return nil, fmt.Errorf("cannot create branch '%s': already exists. Use -c to checkout existing branch", branch)
		}
		if err := m.gitOps.CreateWorktree(worktreePath, branch, baseBranch); err != nil {
			return nil, fmt.Errorf("failed to create worktree with new branch: %w", err)
		}
	case BranchModeCheckout:
		// Use existing branch (fail if doesn't exist)
		if !exists {
			return nil, fmt.Errorf("cannot checkout '%s': branch does not exist. Use -b to create new branch", branch)
		}
		if err := m.gitOps.CreateWorktreeFromExistingBranch(worktreePath, branch); err != nil {
			return nil, fmt.Errorf("failed to create worktree from existing branch: %w", err)
		}
	default:
		// Should never happen as BranchModeCreate is the default
		return nil, fmt.Errorf("invalid branch mode: %v", opts.BranchMode)
	}

	// Create workspace metadata
	workspace := &Workspace{
		ID:          id,
		Name:        opts.Name,
		Branch:      branch,
		BaseBranch:  baseBranch,
		Path:        worktreePath,
		Description: opts.Description,
		StoragePath: filepath.Join(workspaceDir, "storage"),
		CreatedAt:   time.Now(),
		AutoCreated: opts.AutoCreated,
	}

	// Create storage directory
	if err := os.MkdirAll(workspace.StoragePath, 0o755); err != nil {
		// Cleanup on failure
		_ = m.gitOps.RemoveWorktree(worktreePath)
		_ = m.gitOps.DeleteBranch(branch)
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Save workspace metadata
	if err := m.saveWorkspace(ctx, workspace); err != nil {
		// Cleanup on failure
		_ = m.gitOps.RemoveWorktree(worktreePath)
		_ = m.gitOps.DeleteBranch(branch)
		return nil, fmt.Errorf("failed to save workspace metadata: %w", err)
	}

	// Generate and assign index
	index, err := m.idMapper.AddWorkspace(workspace.ID)
	if err != nil {
		// Don't fail if index generation fails, just log it
		// The workspace is already created successfully
		workspace.Index = ""
	} else {
		workspace.Index = index
	}

	// No template files needed - workspaces are clean git worktrees

	return workspace, nil
}

// Get retrieves a workspace by its full ID
func (m *Manager) Get(ctx context.Context, id ID) (*Workspace, error) {
	workspaceMetaPath := filepath.Join(m.workspacesDir, string(id), "workspace.yaml")

	workspace, _, err := m.fm.Read(ctx, workspaceMetaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("workspace not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read workspace: %w", err)
	}

	// Get last modified time from filesystem
	workspace.UpdatedAt = m.getLastModified(workspace.Path)

	// Populate index
	if index, exists := m.idMapper.GetWorkspaceIndex(workspace.ID); exists {
		workspace.Index = index
	}

	// Check consistency status
	m.CheckConsistency(workspace)

	return workspace, nil
}

// List returns all workspaces
func (m *Manager) List(ctx context.Context, opts ListOptions) ([]*Workspace, error) {
	// Ensure workspaces directory exists
	if err := os.MkdirAll(m.workspacesDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create workspaces directory: %w", err)
	}

	files, err := os.ReadDir(m.workspacesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspaces directory: %w", err)
	}

	var workspaces []*Workspace
	var existingIDs []string
	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		// Look for workspace.yaml inside the directory
		workspaceMetaPath := filepath.Join(m.workspacesDir, file.Name(), "workspace.yaml")
		workspacePtr, _, err := m.fm.Read(ctx, workspaceMetaPath)
		if err != nil {
			continue
		}
		workspace := *workspacePtr

		// Collect existing IDs for reconciliation
		existingIDs = append(existingIDs, workspace.ID)

		// Get last modified time from filesystem
		workspace.UpdatedAt = m.getLastModified(workspace.Path)

		// Populate index
		if index, exists := m.idMapper.GetWorkspaceIndex(workspace.ID); exists {
			workspace.Index = index
		}

		// Check consistency status
		m.CheckConsistency(&workspace)

		workspaces = append(workspaces, &workspace)
	}

	// Reconcile index state with actual workspaces
	orphanedCount, err := m.idMapper.ReconcileWorkspaces(existingIDs)
	if err != nil {
		// Log error but continue - don't fail the list operation
		_ = err // Ignore error to satisfy linter
	} else if orphanedCount > 0 {
		// Successfully cleaned up orphaned indices
		// In production, this would be logged at debug level
		_ = orphanedCount // Reference to satisfy linter
	}

	// Sort by creation time (newest first)
	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].CreatedAt.After(workspaces[j].CreatedAt)
	})

	return workspaces, nil
}

// getLastModified gets the last modified time of any file in the workspace
func (m *Manager) getLastModified(workspacePath string) time.Time {
	var lastMod time.Time

	// Walk through the workspace directory
	err := filepath.Walk(workspacePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // Continue walking even if there's an error
		}

		// Skip .git directory and .amux directory
		if strings.Contains(path, "/.git/") || strings.Contains(path, "/.amux/") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Update last modified time if this file is newer
		if info.ModTime().After(lastMod) {
			lastMod = info.ModTime()
		}

		return nil
	})

	// If no files found or error, use directory's own modified time
	if err != nil || lastMod.IsZero() {
		if info, err := os.Stat(workspacePath); err == nil {
			lastMod = info.ModTime()
		}
	}

	return lastMod
}

// ResolveWorkspace finds a workspace by index, full ID, or name
func (m *Manager) ResolveWorkspace(ctx context.Context, identifier Identifier) (*Workspace, error) {
	// 1. Try as full ID
	ws, err := m.Get(ctx, ID(identifier))
	if err == nil {
		return ws, nil
	}

	// 2. Try as index
	if fullID, exists := m.idMapper.GetWorkspaceFull(string(identifier)); exists {
		ws, err = m.Get(ctx, ID(fullID))
		if err == nil {
			return ws, nil
		}
	}

	// 3. Try as name
	workspaces, err := m.List(ctx, ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	var matches []*Workspace
	for _, ws := range workspaces {
		if ws.Name == string(identifier) {
			matches = append(matches, ws)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("workspace not found: %s", identifier)
	case 1:
		return matches[0], nil
	default:
		// Multiple workspaces with the same name
		return nil, fmt.Errorf("multiple workspaces found with name '%s', please use ID or index instead", identifier)
	}
}

// Remove deletes a workspace by identifier (ID, index, or name)
func (m *Manager) Remove(ctx context.Context, identifier Identifier) error {
	workspace, err := m.ResolveWorkspace(ctx, identifier)
	if err != nil {
		return err
	}

	// Check consistency to determine the right cleanup approach
	m.CheckConsistency(workspace)

	// Handle different inconsistency cases
	switch workspace.Status {
	case StatusConsistent:
		// Normal case: both worktree and folder exist
		if err := m.gitOps.RemoveWorktree(workspace.Path); err != nil {
			return fmt.Errorf("failed to remove worktree: %w", err)
		}
	case StatusFolderMissing:
		// Case 1: Folder deleted but git worktree exists
		// Need to prune the worktree reference
		_ = m.gitOps.PruneWorktrees()
	case StatusWorktreeMissing:
		// Case 2: Git worktree removed but folder exists
		// Nothing special needed, will clean up folder below
	case StatusOrphaned:
		// Both are missing, just clean up metadata
	}

	// Try to remove git worktree if it exists (for backward compatibility)
	if workspace.WorktreeExists {
		if err := m.gitOps.RemoveWorktree(workspace.Path); err != nil {
			// If it fails, try pruning
			_ = m.gitOps.PruneWorktrees()
		}
	}

	// Delete branch
	if err := m.gitOps.DeleteBranch(workspace.Branch); err != nil {
		// If branch doesn't exist or is checked out in a non-existent worktree, continue
		if !strings.Contains(err.Error(), "not found") &&
			!strings.Contains(err.Error(), "checked out at") {
			return fmt.Errorf("failed to delete branch: %w", err)
		}
	}

	// Remove index mapping
	_ = m.idMapper.RemoveWorkspace(workspace.ID)

	// Clean up entire workspace directory (which contains worktree and workspace.yaml)
	workspaceDir := filepath.Join(m.workspacesDir, workspace.ID)
	if _, err := os.Stat(workspaceDir); err == nil {
		if err := os.RemoveAll(workspaceDir); err != nil {
			return fmt.Errorf("failed to remove workspace directory: %w", err)
		}
	}

	return nil
}

// Cleanup removes old workspaces based on last modified time
func (m *Manager) Cleanup(ctx context.Context, opts CleanupOptions) ([]string, error) {
	workspaces, err := m.List(ctx, ListOptions{})
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().AddDate(0, 0, -opts.Days)
	var removed []string

	for _, workspace := range workspaces {
		if workspace.UpdatedAt.Before(cutoff) {
			if !opts.DryRun {
				if err := m.Remove(ctx, Identifier(workspace.ID)); err != nil {
					// Log error but continue with other workspaces
					fmt.Fprintf(os.Stderr, "Failed to remove workspace %s: %v\n", workspace.ID, err)
					continue
				}
			}
			removed = append(removed, workspace.ID)
		}
	}

	return removed, nil
}

// CheckConsistency checks the consistency status of a workspace
func (m *Manager) CheckConsistency(workspace *Workspace) {
	// Check if workspace folder exists
	if _, err := os.Stat(workspace.Path); err == nil {
		workspace.PathExists = true
	} else {
		workspace.PathExists = false
	}

	// Check if git worktree exists
	worktrees, err := m.gitOps.ListWorktrees()
	workspace.WorktreeExists = false
	if err == nil {
		for _, wt := range worktrees {
			// Normalize paths for comparison
			wtPath := filepath.Clean(wt.Path)
			wsPath := filepath.Clean(workspace.Path)

			// Try to resolve symlinks for both paths to handle macOS /var -> /private/var
			// We need to resolve even non-existent paths by resolving their parent directories
			wtPathResolved := wtPath
			wsPathResolved := wsPath

			// For worktree path
			if resolvedWt, err := filepath.EvalSymlinks(wtPath); err == nil {
				wtPathResolved = resolvedWt
			} else {
				// If path doesn't exist, try to resolve parent directory
				wtDir := filepath.Dir(wtPath)
				if resolvedDir, err := filepath.EvalSymlinks(wtDir); err == nil {
					wtPathResolved = filepath.Join(resolvedDir, filepath.Base(wtPath))
				}
			}

			// For workspace path
			if resolvedWs, err := filepath.EvalSymlinks(wsPath); err == nil {
				wsPathResolved = resolvedWs
			} else {
				// If path doesn't exist, try to resolve parent directory
				wsDir := filepath.Dir(wsPath)
				if resolvedDir, err := filepath.EvalSymlinks(wsDir); err == nil {
					wsPathResolved = filepath.Join(resolvedDir, filepath.Base(wsPath))
				}
			}

			if wtPathResolved == wsPathResolved {
				workspace.WorktreeExists = true
				break
			}
		}
	}

	// Determine status based on existence flags
	if workspace.PathExists && workspace.WorktreeExists {
		workspace.Status = StatusConsistent
	} else if !workspace.PathExists && workspace.WorktreeExists {
		workspace.Status = StatusFolderMissing
	} else if workspace.PathExists && !workspace.WorktreeExists {
		workspace.Status = StatusWorktreeMissing
	} else {
		workspace.Status = StatusOrphaned
	}
}

// saveWorkspace saves workspace metadata to disk with file locking
func (m *Manager) saveWorkspace(ctx context.Context, workspace *Workspace) error {
	// Save workspace.yaml inside the workspace directory (not in the worktree)
	workspaceDir := filepath.Join(m.workspacesDir, workspace.ID)
	workspaceMetaPath := filepath.Join(workspaceDir, "workspace.yaml")

	// Ensure workspace directory exists
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	return m.fm.Write(ctx, workspaceMetaPath, workspace)
}

// generateID generates a unique workspace ID
func generateID(name string) string {
	// Sanitize name for use in ID
	safeName := strings.ToLower(name)
	safeName = strings.ReplaceAll(safeName, " ", "-")
	safeName = strings.ReplaceAll(safeName, "_", "-")

	// Keep only alphanumeric and hyphens
	var cleaned []rune
	for _, r := range safeName {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			cleaned = append(cleaned, r)
		}
	}
	safeName = string(cleaned)

	// Limit length
	if len(safeName) > 20 {
		safeName = safeName[:20]
	}

	// Generate unique suffix
	timestamp := time.Now().Unix()
	randomSuffix := uuid.New().String()[:8]

	return fmt.Sprintf("workspace-%s-%d-%s", safeName, timestamp, randomSuffix)
}

// GetIDMapper returns the ID mapper used by the workspace manager
func (m *Manager) GetIDMapper() *idmap.IDMapper {
	return m.idMapper
}
