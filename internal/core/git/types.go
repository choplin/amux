package git

// WorktreeInfo represents information about a git worktree
type WorktreeInfo struct {
	Path   string
	Branch string
	Commit string
}

// RepositoryInfo represents information about a git repository
type RepositoryInfo struct {
	Path          string
	CurrentBranch string
	RemoteURL     string
	IsClean       bool
}