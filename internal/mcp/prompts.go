package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/aki/amux/internal/workspace"
)

// registerPrompts registers all MCP prompts
func (s *ServerV2) registerPrompts() error {
	// Register start-issue-work prompt
	startIssuePrompt := mcp.NewPrompt(
		"start-issue-work",
		mcp.WithPromptDescription("Guide through starting work on a GitHub issue. Helps AI agents properly understand requirements before starting implementation"),
		mcp.WithArgument("issue_number",
			mcp.ArgumentDescription("GitHub issue number to work on"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("issue_title",
			mcp.ArgumentDescription("Title of the GitHub issue"),
		),
		mcp.WithArgument("issue_url",
			mcp.ArgumentDescription("Full URL to the GitHub issue"),
		),
	)
	s.mcpServer.AddPrompt(startIssuePrompt, s.handleStartIssueWorkPrompt)

	// Register prepare-pr prompt
	preparePRPrompt := mcp.NewPrompt(
		"prepare-pr",
		mcp.WithPromptDescription("Guide through preparing a pull request. Helps ensure all tests pass and code is properly formatted before creating a PR"),
		mcp.WithArgument("workspace_id",
			mcp.ArgumentDescription("Workspace ID or name to prepare PR from"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("pr_title",
			mcp.ArgumentDescription("Proposed PR title"),
		),
		mcp.WithArgument("pr_description",
			mcp.ArgumentDescription("Proposed PR description"),
		),
	)
	s.mcpServer.AddPrompt(preparePRPrompt, s.handlePreparePRPrompt)

	// Register review-workspace prompt
	reviewWorkspacePrompt := mcp.NewPrompt(
		"review-workspace",
		mcp.WithPromptDescription("Review workspace state and suggest next steps. Analyzes the current workspace state and provides guidance on what to do next"),
		mcp.WithArgument("workspace_id",
			mcp.ArgumentDescription("Workspace ID or name to review"),
			mcp.RequiredArgument(),
		),
	)
	s.mcpServer.AddPrompt(reviewWorkspacePrompt, s.handleReviewWorkspacePrompt)

	return nil
}

// handleStartIssueWorkPrompt guides through starting work on an issue
func (s *ServerV2) handleStartIssueWorkPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	// Extract arguments from request

	issueNumber, ok := request.Params.Arguments["issue_number"]
	if !ok || issueNumber == "" {
		return nil, fmt.Errorf("issue_number is required")
	}

	issueTitle := request.Params.Arguments["issue_title"]
	issueURL := request.Params.Arguments["issue_url"]

	// Build the prompt messages
	messages := []mcp.PromptMessage{
		{
			Role: mcp.RoleUser,
			Content: mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("I need to work on issue #%s", issueNumber),
			},
		},
		{
			Role: mcp.RoleAssistant,
			Content: mcp.TextContent{
				Type: "text",
				Text: s.buildStartIssueWorkPromptText(issueNumber, issueTitle, issueURL),
			},
		},
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Guidance for starting work on issue #%s", issueNumber),
		Messages:    messages,
	}, nil
}

// buildStartIssueWorkPromptText creates the prompt text for starting issue work
func (s *ServerV2) buildStartIssueWorkPromptText(issueNumber, issueTitle, issueURL string) string {
	var sb strings.Builder

	sb.WriteString("I'll help you start working on this issue. Let's ensure we fully understand the requirements.\n\n")

	sb.WriteString("## 🔍 Requirements Clarification\n\n")
	sb.WriteString("Before we begin implementation, let's make sure we understand:\n\n")
	sb.WriteString("1. **What is the problem?** - What specific issue are we solving?\n")
	sb.WriteString("2. **Who is affected?** - Which users or components are impacted?\n")
	sb.WriteString("3. **What is the desired outcome?** - What should the solution achieve?\n")
	sb.WriteString("4. **Are there constraints?** - Technical limitations, backwards compatibility, etc.\n")
	sb.WriteString("5. **How will we test it?** - What tests or validation are needed?\n\n")

	if issueURL != "" {
		sb.WriteString(fmt.Sprintf("📎 Issue URL: %s\n\n", issueURL))
	}

	sb.WriteString("## 📋 Recommended Workflow\n\n")
	sb.WriteString("1. **Read the issue** thoroughly, including all comments\n")
	sb.WriteString("2. **Ask clarifying questions** if anything is unclear\n")
	sb.WriteString("3. **Create a workspace** for this issue:\n")
	sb.WriteString("   ```\n")
	sb.WriteString("   workspace_create({\n")
	sb.WriteString(fmt.Sprintf("     name: \"issue-%s\",\n", issueNumber))
	if issueTitle != "" {
		sb.WriteString(fmt.Sprintf("     description: \"%s\",\n", issueTitle))
	}
	sb.WriteString("     baseBranch: \"main\"\n")
	sb.WriteString("   })\n")
	sb.WriteString("   ```\n")
	sb.WriteString("4. **Document your understanding** in the workspace context.md\n")
	sb.WriteString("5. **Create an implementation plan** before coding\n")
	sb.WriteString("6. **Implement incrementally** with regular testing\n\n")

	sb.WriteString("## 💡 Tips\n\n")
	sb.WriteString("- Don't rush into coding - understanding is crucial\n")
	sb.WriteString("- Break complex issues into smaller tasks\n")
	sb.WriteString("- Test early and often\n")
	sb.WriteString("- Keep the PR focused on the specific issue\n\n")

	sb.WriteString("**Ready to proceed?** Start by reading the issue and let me know if you need clarification!")

	return sb.String()
}

// handlePreparePRPrompt guides through preparing a pull request
func (s *ServerV2) handlePreparePRPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	// Extract arguments from request

	workspaceID, ok := request.Params.Arguments["workspace_id"]
	if !ok || workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}

	// Resolve workspace by ID or name
	ws, err := s.workspaceManager.ResolveWorkspace(ctx, workspace.Identifier(workspaceID))
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	prTitle := request.Params.Arguments["pr_title"]
	prDescription := request.Params.Arguments["pr_description"]

	// Build the prompt messages
	messages := []mcp.PromptMessage{
		{
			Role: mcp.RoleUser,
			Content: mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("I need to prepare a PR from workspace %s", ws.Name),
			},
		},
		{
			Role: mcp.RoleAssistant,
			Content: mcp.TextContent{
				Type: "text",
				Text: s.buildPreparePRPromptText(ws, prTitle, prDescription),
			},
		},
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Guidance for preparing PR from workspace %s", ws.Name),
		Messages:    messages,
	}, nil
}

// buildPreparePRPromptText creates the prompt text for preparing a PR
func (s *ServerV2) buildPreparePRPromptText(ws *workspace.Workspace, prTitle, prDescription string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Let's prepare a pull request from workspace **%s**.\n\n", ws.Name))

	sb.WriteString("## ✅ Pre-PR Checklist\n\n")
	sb.WriteString("Before creating the PR, ensure:\n\n")
	sb.WriteString("1. **All changes are committed**\n")
	sb.WriteString("   ```bash\n")
	sb.WriteString("   git status\n")
	sb.WriteString("   git add -A\n")
	sb.WriteString("   git commit -m \"Your commit message\"\n")
	sb.WriteString("   ```\n\n")

	sb.WriteString("2. **Tests pass**\n")
	sb.WriteString("   ```bash\n")
	sb.WriteString("   go test ./...\n")
	sb.WriteString("   ```\n\n")

	sb.WriteString("3. **Code is properly formatted**\n")
	sb.WriteString("   ```bash\n")
	sb.WriteString("   go fmt ./...\n")
	sb.WriteString("   goimports -w .\n")
	sb.WriteString("   ```\n\n")

	sb.WriteString("4. **Linting passes**\n")
	sb.WriteString("   ```bash\n")
	sb.WriteString("   golangci-lint run\n")
	sb.WriteString("   ```\n\n")

	sb.WriteString("5. **Branch is up to date**\n")
	sb.WriteString("   ```bash\n")
	sb.WriteString(fmt.Sprintf("   git fetch origin %s\n", ws.BaseBranch))
	sb.WriteString(fmt.Sprintf("   git rebase origin/%s\n", ws.BaseBranch))
	sb.WriteString("   ```\n\n")

	sb.WriteString("## 📝 PR Preparation\n\n")

	if prTitle == "" {
		sb.WriteString("### Title\n")
		sb.WriteString("Create a clear, descriptive title that:\n")
		sb.WriteString("- Starts with a conventional commit type (feat, fix, chore, etc.)\n")
		sb.WriteString("- Briefly describes what the PR does\n")
		sb.WriteString("- References the issue number if applicable\n\n")
		sb.WriteString("Example: `feat(mcp): implement resource architecture (#44)`\n\n")
	}

	if prDescription == "" {
		sb.WriteString("### Description\n")
		sb.WriteString("Write a comprehensive description that includes:\n")
		sb.WriteString("- **What** - What changes does this PR introduce?\n")
		sb.WriteString("- **Why** - Why are these changes needed?\n")
		sb.WriteString("- **How** - Brief overview of the implementation\n")
		sb.WriteString("- **Testing** - How were the changes tested?\n")
		sb.WriteString("- **Screenshots** - If UI changes, include before/after\n\n")
	}

	sb.WriteString("## 🚀 Creating the PR\n\n")
	sb.WriteString("Once everything is ready:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Push your branch\n")
	sb.WriteString(fmt.Sprintf("git push -u origin %s\n\n", ws.Branch))
	sb.WriteString("# Create PR\n")
	sb.WriteString("gh pr create --draft \\\n")

	if prTitle != "" {
		sb.WriteString(fmt.Sprintf("  --title \"%s\" \\\n", prTitle))
	} else {
		sb.WriteString("  --title \"Your PR title\" \\\n")
	}

	sb.WriteString(fmt.Sprintf("  --base %s \\\n", ws.BaseBranch))
	sb.WriteString("  --body \"Your PR description\"\n")
	sb.WriteString("```\n\n")

	sb.WriteString("💡 **Tip**: Create as draft first to run CI checks before requesting review!")

	return sb.String()
}

// handleReviewWorkspacePrompt reviews workspace state
func (s *ServerV2) handleReviewWorkspacePrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	// Extract arguments from request

	workspaceID, ok := request.Params.Arguments["workspace_id"]
	if !ok || workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}

	// Resolve workspace by ID or name
	ws, err := s.workspaceManager.ResolveWorkspace(ctx, workspace.Identifier(workspaceID))
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	// Build the prompt messages
	messages := []mcp.PromptMessage{
		{
			Role: mcp.RoleUser,
			Content: mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Review the state of workspace %s", ws.Name),
			},
		},
		{
			Role: mcp.RoleAssistant,
			Content: mcp.TextContent{
				Type: "text",
				Text: s.buildReviewWorkspacePromptText(ws),
			},
		},
	}

	return &mcp.GetPromptResult{
		Description: fmt.Sprintf("Review of workspace %s state", ws.Name),
		Messages:    messages,
	}, nil
}

// buildReviewWorkspacePromptText creates the prompt text for reviewing a workspace
func (s *ServerV2) buildReviewWorkspacePromptText(ws *workspace.Workspace) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 📊 Workspace Review: %s\n\n", ws.Name))

	// Workspace metadata
	sb.WriteString("### 📋 Workspace Details\n")
	sb.WriteString(fmt.Sprintf("- **ID**: %s\n", ws.ID))
	sb.WriteString(fmt.Sprintf("- **Branch**: %s\n", ws.Branch))
	sb.WriteString(fmt.Sprintf("- **Base Branch**: %s\n", ws.BaseBranch))
	sb.WriteString(fmt.Sprintf("- **Created**: %s\n", ws.CreatedAt.Format("2006-01-02 15:04:05")))

	// Calculate age
	age := time.Since(ws.CreatedAt)
	if age > 24*time.Hour {
		days := int(age.Hours() / 24)
		sb.WriteString(fmt.Sprintf("- **Age**: %d days\n", days))
		if days > 7 {
			sb.WriteString("  ⚠️ This workspace is over a week old - consider wrapping up or rebasing\n")
		}
	} else {
		sb.WriteString(fmt.Sprintf("- **Age**: %d hours\n", int(age.Hours())))
	}

	if ws.Description != "" {
		sb.WriteString(fmt.Sprintf("- **Description**: %s\n", ws.Description))
	}
	sb.WriteString("\n")

	// Next steps section
	sb.WriteString("### 🎯 Recommended Next Steps\n\n")
	sb.WriteString("To properly review this workspace, I recommend:\n\n")

	sb.WriteString("1. **Check git status**\n")
	sb.WriteString("   ```bash\n")
	sb.WriteString("   git status\n")
	sb.WriteString("   git log --oneline -10\n")
	sb.WriteString("   ```\n\n")

	sb.WriteString("2. **Review changes**\n")
	sb.WriteString("   ```bash\n")
	sb.WriteString(fmt.Sprintf("   git diff %s...HEAD\n", ws.BaseBranch))
	sb.WriteString("   ```\n\n")

	sb.WriteString("3. **Check context**\n")
	sb.WriteString("   - Read `amux://workspace/" + ws.ID + "/context` for workspace context\n")
	sb.WriteString("   - Browse files with `amux://workspace/" + ws.ID + "/files`\n\n")

	sb.WriteString("4. **Run tests**\n")
	sb.WriteString("   ```bash\n")
	sb.WriteString("   go test ./...\n")
	sb.WriteString("   ```\n\n")

	sb.WriteString("### 💭 Questions to Consider\n\n")
	sb.WriteString("- Is the work in this workspace complete?\n")
	sb.WriteString("- Are all tests passing?\n")
	sb.WriteString("- Is the code properly documented?\n")
	sb.WriteString("- Should this be merged or needs more work?\n")
	sb.WriteString("- Are there any uncommitted changes?\n\n")

	sb.WriteString("### 🔄 Possible Actions\n\n")
	sb.WriteString("- **Continue work**: Make additional changes\n")
	sb.WriteString("- **Prepare PR**: Use `prepare-pr` prompt\n")
	sb.WriteString("- **Rebase**: Update from " + ws.BaseBranch + "\n")
	sb.WriteString("- **Archive**: Remove if work is complete or abandoned\n")

	return sb.String()
}
