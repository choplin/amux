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
			`workspace_create(name: "fix-issue-30", description: "Fix authentication bug (#30)")`,
			`workspace_create(name: "feat-dark-mode", description: "Implement dark mode feature")`,
			`workspace_create(name: "refactor-api", baseBranch: "develop", description: "Refactor API structure")`,
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
			`workspace_remove(workspace_identifier: "fix-issue-30")`,
			`workspace_remove(workspace_identifier: "3")`,
			`workspace_remove(workspace_identifier: "workspace-feat-api-1234567890-abcdef12")`,
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
			`resource_workspace_browse(workspace_identifier: "current", path: ".")`,
			`resource_workspace_browse(workspace_identifier: "1", path: "src/")`,
			`resource_workspace_browse(workspace_identifier: "fix-auth", path: "internal/auth")`,
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
			`storage_read(workspace_identifier: "current", path: "README.md")`,
			`storage_read(workspace_identifier: "1", path: "src/main.go")`,
			`storage_read(session_identifier: "session-123", path: "debug.log")`,
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
			`storage_write(workspace_identifier: "current", path: "NOTES.md", content: "# Implementation Plan\n...")`,
			`storage_write(workspace_identifier: "1", path: "src/feature.go", content: "package main\n...")`,
			`storage_write(workspace_identifier: "fix-auth", path: "tests/auth_test.go", content: "...")`,
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
			`session_run(agent_id: "test-runner", command: "npm test", workspace_identifier: "current")`,
			`session_run(agent_id: "builder", command: "go build ./...", workspace_identifier: "1")`,
			`session_run(name: "debug-session", command: "python debug.py", workspace_identifier: "fix-auth")`,
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
			`resource_session_output(session_identifier: "session-abc123")`,
			`resource_session_output(session_identifier: "current")`,
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
			`session_send_input(session_identifier: "session-123", input: "yes\n")`,
			`session_send_input(session_identifier: "current", input: "npm install express\n")`,
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
			`session_stop(session_identifier: "session-123")`,
			`session_stop(session_identifier: "current")`,
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
			`resource_workspace_list()`,
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
			`resource_workspace_show(workspace_identifier: "1")`,
			`resource_workspace_show(workspace_identifier: "fix-auth")`,
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
			`storage_list(workspace_identifier: "current", path: ".")`,
			`storage_list(session_identifier: "session-123", path: "logs/")`,
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
			`resource_session_list()`,
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
			`resource_session_show(session_identifier: "session-123")`,
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
