package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
)

// Operations provides git operations for AgentCave
type Operations struct {
	repoPath string
}

// NewOperations creates a new git operations instance
func NewOperations(repoPath string) *Operations {
	return &Operations{
		repoPath: repoPath,
	}
}

// IsGitRepository checks if the path is a git repository
func (o *Operations) IsGitRepository() bool {
	_, err := git.PlainOpen(o.repoPath)
	return err == nil
}

// GetRepositoryInfo returns information about the repository
func (o *Operations) GetRepositoryInfo() (*RepositoryInfo, error) {
	repo, err := git.PlainOpen(o.repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	info := &RepositoryInfo{
		Path: o.repoPath,
	}

	// Get current branch
	ref, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}
	info.CurrentBranch = ref.Name().Short()

	// Get remote URL
	remotes, err := repo.Remotes()
	if err == nil && len(remotes) > 0 {
		config := remotes[0].Config()
		if len(config.URLs) > 0 {
			info.RemoteURL = config.URLs[0]
		}
	}

	// Check if repository is clean
	wt, err := repo.Worktree()
	if err == nil {
		status, err := wt.Status()
		if err == nil {
			info.IsClean = status.IsClean()
		}
	}

	return info, nil
}

// CreateWorktree creates a new git worktree
func (o *Operations) CreateWorktree(path, branch string, baseBranch string) error {
	// First create the branch from baseBranch
	if err := o.CreateBranch(branch, baseBranch); err != nil {
		return err
	}

	// Use git command for worktree operations (go-git doesn't fully support worktrees)
	cmd := exec.Command("git", "worktree", "add", path, branch)
	cmd.Dir = o.repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create worktree: %s", output)
	}

	return nil
}

// CreateWorktreeFromExistingBranch creates a worktree from an existing branch
func (o *Operations) CreateWorktreeFromExistingBranch(path, branch string) error {
	// Use git command for worktree operations (go-git doesn't fully support worktrees)
	cmd := exec.Command("git", "worktree", "add", path, branch)
	cmd.Dir = o.repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create worktree from existing branch %s: %s", branch, output)
	}

	return nil
}

// RemoveWorktree removes a git worktree
func (o *Operations) RemoveWorktree(path string) error {
	cmd := exec.Command("git", "worktree", "remove", "--force", path)
	cmd.Dir = o.repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %s", output)
	}

	return nil
}

// CreateBranch creates a new branch from a base branch
func (o *Operations) CreateBranch(branch, baseBranch string) error {
	cmd := exec.Command("git", "branch", branch, baseBranch)
	cmd.Dir = o.repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if branch already exists
		if strings.Contains(string(output), "already exists") {
			return nil
		}
		return fmt.Errorf("failed to create branch: %s", output)
	}

	return nil
}

// DeleteBranch deletes a branch
func (o *Operations) DeleteBranch(branch string) error {
	cmd := exec.Command("git", "branch", "-D", branch)
	cmd.Dir = o.repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete branch: %s", output)
	}

	return nil
}

// ListWorktrees lists all worktrees in the repository
func (o *Operations) ListWorktrees() ([]*WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = o.repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return parseWorktreeList(output), nil
}

// GetDefaultBranch returns the default branch name (main or master)
func (o *Operations) GetDefaultBranch() (string, error) {
	// Try to get the default branch from remote
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = o.repoPath

	output, err := cmd.Output()
	if err == nil {
		branch := strings.TrimSpace(string(output))
		branch = strings.TrimPrefix(branch, "refs/remotes/origin/")
		if branch != "" {
			return branch, nil
		}
	}

	// Fallback: check if main or master exists
	branches := []string{"main", "master"}
	for _, branch := range branches {
		cmd := exec.Command("git", "rev-parse", "--verify", branch)
		cmd.Dir = o.repoPath
		if err := cmd.Run(); err == nil {
			return branch, nil
		}
	}

	return "main", nil
}

// parseWorktreeList parses the output of 'git worktree list --porcelain'
func parseWorktreeList(output []byte) []*WorktreeInfo {
	var worktrees []*WorktreeInfo
	lines := bytes.Split(output, []byte("\n"))

	var current *WorktreeInfo
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			if current != nil {
				worktrees = append(worktrees, current)
				current = nil
			}
			continue
		}

		parts := bytes.SplitN(line, []byte(" "), 2)
		if len(parts) != 2 {
			continue
		}

		key := string(parts[0])
		value := string(parts[1])

		switch key {
		case "worktree":
			current = &WorktreeInfo{Path: value}
		case "branch":
			if current != nil {
				current.Branch = strings.TrimPrefix(value, "refs/heads/")
			}
		case "HEAD":
			if current != nil {
				current.Commit = value
			}
		}
	}

	if current != nil {
		worktrees = append(worktrees, current)
	}

	return worktrees
}

// ValidateWorktreePath ensures the worktree path is safe and within project bounds
func ValidateWorktreePath(basePath, requestedPath string) error {
	// Ensure the base path is absolute
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("failed to resolve base path: %w", err)
	}

	// Resolve the requested path relative to base
	fullPath := filepath.Join(absBase, requestedPath)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Ensure the path is within the base directory
	if !strings.HasPrefix(absPath, absBase+string(os.PathSeparator)) && absPath != absBase {
		return fmt.Errorf("path is outside workspace boundaries")
	}

	return nil
}
