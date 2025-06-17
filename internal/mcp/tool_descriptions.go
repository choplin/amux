package mcp

// ToolDescription provides enhanced descriptions for AI agents
type ToolDescription struct {
	Description string
	WhenToUse   []string
	Examples    []string
	NextTools   []string
}

// Enhanced tool descriptions for better AI discoverability
var toolDescriptions = map[string]ToolDescription{
	"workspace_create": {
		Description: "Create a new isolated git worktree-based workspace for development. Each workspace has its own branch and can be used for working on separate features or issues",
		WhenToUse: []string{
			"When starting work on a new feature or bug fix",
			"When you need an isolated environment to test changes",
			"When asked to 'work on issue #X' or 'implement feature Y'",
			"Before making any code changes to the repository",
			"When you want to experiment without affecting the main branch",
		},
		Examples: []string{
			`workspace_create(name: "fix-issue-30") → {id: "workspace-fix-issue-30-...", name: "fix-issue-30", branch: "fix-issue-30"}`,
			`workspace_create(name: "feat-api", description: "New API endpoints") → {id: "workspace-feat-api-...", name: "feat-api", description: "New API endpoints"}`,
			`workspace_create(name: "hotfix", baseBranch: "release/v2") → {id: "workspace-hotfix-...", branch: "hotfix", base_branch: "release/v2"}`,
		},
		NextTools: []string{
			"resource_workspace_browse - Explore the workspace structure",
			"storage_write - Document your plan or findings",
			"session_run - Run tests or build commands",
		},
	},

	"workspace_remove": {
		Description: "Remove a workspace and its associated git worktree. This permanently deletes the workspace directory and cannot be undone",
		WhenToUse: []string{
			"After a pull request has been merged",
			"When cleaning up abandoned or experimental workspaces",
			"When explicitly asked to remove or delete a workspace",
			"After completing work on a feature or fix",
		},
		Examples: []string{
			`workspace_remove(workspace_identifier: "fix-issue-30") → {message: "Workspace fix-issue-30 removed"}`,
			`workspace_remove(workspace_identifier: "3") → {message: "Workspace feat-api (3) removed"}`,
		},
		NextTools: []string{
			"resource_workspace_list - Verify remaining workspaces",
			"workspace_create - Create a new workspace for the next task",
		},
	},

	"resource_workspace_browse": {
		Description: "Browse files in a workspace (replaces ls, find, tree commands). Returns directory listings or file contents. FASTER than bash commands and provides better context",
		WhenToUse: []string{
			"When you need to explore project structure",
			"When looking for specific files or understanding code organization",
			"Instead of using bash commands like 'ls', 'find', or 'tree'",
			"Before making changes to understand existing code patterns",
			"When asked 'what files are in X directory?'",
			"To check if a file or directory exists",
		},
		Examples: []string{
			`resource_workspace_browse(workspace_identifier: "1") → {content: "src/\nREADME.md\ngo.mod\n..."}`,
			`resource_workspace_browse(workspace_identifier: "1", path: "src/") → {content: "main.go\nconfig/\nhandlers/\n..."}`,
			`resource_workspace_browse(workspace_identifier: "1", path: "README.md") → {content: "# Project Title\n\nDescription..."}`,
		},
		NextTools: []string{
			"storage_read - Read the full content of files you found",
			"storage_write - Create new files or document findings",
			"session_run - Execute commands in the workspace",
		},
	},

	"storage_read": {
		Description: "Read files from workspace or session storage (replaces cat, head, tail commands). Use this instead of bash file reading commands. Requires either workspace_identifier or session_identifier",
		WhenToUse: []string{
			"When you need to read file contents",
			"Instead of using bash commands like 'cat', 'head', or 'tail'",
			"To examine source code, configuration files, or documentation",
			"After using workspace_browse to find files",
			"When implementing features to understand existing code",
		},
		Examples: []string{
			`storage_read(workspace_identifier: "1", path: "config.yaml") → {content: "name: myapp\nversion: 1.0\n...", size: 256}`,
			`storage_read(session_identifier: "2", path: "output.log") → {content: "[INFO] Starting...\n[ERROR] Failed...", size: 1024}`,
		},
		NextTools: []string{
			"storage_write - Modify the file or create related files",
			"resource_workspace_browse - Find related files",
			"session_run - Test your understanding with commands",
		},
	},

	"storage_write": {
		Description: "Write files to workspace or session storage (replaces echo >, cat >, file creation). Use this for creating or updating files",
		WhenToUse: []string{
			"When creating new files or modifying existing ones",
			"Instead of using bash commands like 'echo >', 'cat >', or text editors",
			"To save implementation code, documentation, or notes",
			"To create test files or configuration",
			"When documenting your findings or creating TODO lists",
		},
		Examples: []string{
			`storage_write(workspace_identifier: "1", path: "NOTES.md", content: "# TODOs\n- Fix auth\n- Add tests") → {path: "NOTES.md", bytes: 28}`,
			`storage_write(session_identifier: "2", path: "results.json", content: "{...}") → {path: "results.json", bytes: 156}`,
		},
		NextTools: []string{
			"session_run - Test your changes",
			"storage_read - Verify the file was written correctly",
			"resource_session_output - Check test or build results",
		},
	},

	"session_run": {
		Description: "Run an AI agent session in a workspace. Creates and immediately starts the session. Better than direct bash commands as it provides session management",
		WhenToUse: []string{
			"When you need to execute commands like tests, builds, or scripts",
			"To run development tools (npm, go, python, etc.)",
			"When testing your implementation",
			"To debug issues by running diagnostic commands",
			"Instead of trying to use bash directly",
		},
		Examples: []string{
			`session_run(agent_id: "test", command: "go test", workspace_identifier: "1") → {id: "session-123", status: "running", tmux_session: "amux-session-123"}`,
			`session_run(agent_id: "shell", workspace_identifier: "2") → {id: "session-124", status: "running", command: "/bin/bash"}`,
			`session_run(agent_id: "custom", name: "build", command: "make", workspace_identifier: "3") → {id: "session-125", name: "build", status: "running"}`,
			`session_run(agent_id: "dev", workspace_identifier: "4", shell: "/bin/zsh", window_name: "development") → {id: "session-126", status: "running"}`,
		},
		NextTools: []string{
			"resource_session_output - Monitor the command output",
			"session_send_input - Send input if the command is interactive",
			"storage_write - Save important output or results",
			"session_stop - Stop the session when done",
		},
	},

	"resource_session_output": {
		Description: "Read session output/logs. Essential for monitoring command execution and debugging",
		WhenToUse: []string{
			"After running session_run to see command results",
			"To check for errors or test failures",
			"To monitor long-running processes",
			"When debugging issues to see detailed output",
			"To verify that commands completed successfully",
		},
		Examples: []string{
			`resource_session_output(session_identifier: "session-123") → {content: "Running tests...\nPASS: auth_test.go\nPASS: main_test.go\n"}`,
			`resource_session_output(session_identifier: "1") → {content: "[ERROR] Build failed: undefined variable\n"}`,
		},
		NextTools: []string{
			"session_send_input - Send commands if errors need fixing",
			"storage_write - Document important findings",
			"session_stop - Stop the session if needed",
		},
	},

	"session_send_input": {
		Description: "Send input text to a running agent session's stdin. Use for interactive commands",
		WhenToUse: []string{
			"When a session is waiting for user input",
			"To answer prompts from interactive commands",
			"To send commands to a REPL or shell session",
			"When debugging interactively",
		},
		Examples: []string{
			`session_send_input(session_identifier: "session-123", input: "yes\n") → {message: "Input sent to session session-123"}`,
			`session_send_input(session_identifier: "2", input: "exit\n") → {message: "Input sent to session 2"}`,
		},
		NextTools: []string{
			"resource_session_output - Check the response",
			"session_stop - Stop when done with interaction",
		},
	},

	"session_stop": {
		Description: "Stop a running agent session gracefully",
		WhenToUse: []string{
			"When a command has completed its task",
			"To clean up long-running or stuck sessions",
			"Before removing a workspace",
			"When explicitly asked to stop a session",
		},
		Examples: []string{
			`session_stop(session_identifier: "session-123") → {message: "Session session-123 stopped successfully"}`,
			`session_stop(session_identifier: "1") → {message: "Session 1 stopped successfully"}`,
		},
		NextTools: []string{
			"resource_session_list - Check remaining sessions",
			"workspace_remove - Clean up the workspace if done",
		},
	},

	"resource_workspace_list": {
		Description: "List all workspaces. Shows ID, name, branch, and other details",
		WhenToUse: []string{
			"To see all available workspaces",
			"When starting work to choose or create a workspace",
			"To find a specific workspace by name or ID",
			"To check workspace status before operations",
		},
		Examples: []string{
			`resource_workspace_list() → [{id: "workspace-fix-auth-...", name: "fix-auth", branch: "fix-auth"}, {id: "workspace-feat-api-...", name: "feat-api", branch: "feat-api"}]`,
		},
		NextTools: []string{
			"resource_workspace_show - Get details of a specific workspace",
			"workspace_create - Create a new workspace if needed",
			"resource_workspace_browse - Explore a workspace",
		},
	},

	"resource_workspace_show": {
		Description: "Get detailed information about a specific workspace",
		WhenToUse: []string{
			"To get the full path of a workspace",
			"To check workspace metadata and status",
			"To verify workspace configuration",
			"Before performing operations on a workspace",
		},
		Examples: []string{
			`resource_workspace_show(workspace_identifier: "1") → {id: "workspace-fix-auth-...", name: "fix-auth", worktree_path: "/path/to/worktree", branch: "fix-auth"}`,
			`resource_workspace_show(workspace_identifier: "feat-api") → {id: "workspace-feat-api-...", name: "feat-api", description: "API refactoring", created_at: "2024-01-15T10:30:00Z"}`,
		},
		NextTools: []string{
			"resource_workspace_browse - Explore the workspace files",
			"session_run - Execute commands in the workspace",
		},
	},

	"storage_list": {
		Description: "List files in workspace or session storage (replaces ls command)",
		WhenToUse: []string{
			"To see what files exist in storage",
			"Instead of using 'ls' command",
			"To check available files before reading",
			"To verify files were created correctly",
		},
		Examples: []string{
			`storage_list(workspace_identifier: "1") → {files: ["README.md", "config.yaml", "src/"], count: 3}`,
			`storage_list(session_identifier: "2", path: "logs/") → {files: ["error.log", "debug.log"], count: 2}`,
		},
		NextTools: []string{
			"storage_read - Read specific files",
			"storage_write - Create new files",
		},
	},

	"resource_session_list": {
		Description: "List all active sessions with their status",
		WhenToUse: []string{
			"To see what sessions are running",
			"To find a specific session ID",
			"To check session status (busy/idle/stuck)",
			"Before creating new sessions",
		},
		Examples: []string{
			`resource_session_list() → [{id: "session-123", name: "test", status: "running", workspace_id: "1"}, {id: "session-124", status: "idle", workspace_id: "2"}]`,
		},
		NextTools: []string{
			"resource_session_show - Get session details",
			"resource_session_output - Check session output",
			"session_stop - Stop unneeded sessions",
		},
	},

	"resource_session_show": {
		Description: "Get detailed information about a specific session",
		WhenToUse: []string{
			"To check session configuration",
			"To verify session status and health",
			"To get session metadata",
		},
		Examples: []string{
			`resource_session_show(session_identifier: "session-123") → {id: "session-123", agent_id: "test", status: "running", command: "go test", started_at: "2024-01-15T10:35:00Z"}`,
		},
		NextTools: []string{
			"resource_session_output - Check session logs",
			"session_send_input - Interact with the session",
			"session_stop - Stop if needed",
		},
	},
}

// GetEnhancedDescription returns the enhanced description for a tool
func GetEnhancedDescription(toolName string) string {
	if desc, ok := toolDescriptions[toolName]; ok {
		result := desc.Description + "\n\nWHEN TO USE THIS TOOL:\n"
		for _, when := range desc.WhenToUse {
			result += "- " + when + "\n"
		}

		if len(desc.Examples) > 0 {
			result += "\nEXAMPLES:\n"
			for _, example := range desc.Examples {
				result += example + "\n"
			}
		}

		return result
	}
	return ""
}

// GetNextToolSuggestions returns suggested next tools for a given tool
func GetNextToolSuggestions(toolName string) []map[string]string {
	if desc, ok := toolDescriptions[toolName]; ok {
		suggestions := make([]map[string]string, 0, len(desc.NextTools))
		for _, next := range desc.NextTools {
			suggestions = append(suggestions, map[string]string{
				"tool": next,
			})
		}
		return suggestions
	}
	return nil
}
