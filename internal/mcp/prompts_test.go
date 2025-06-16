package mcp

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/core/workspace"
)

// convertToStringMap converts map[string]interface{} to map[string]string
func convertToStringMap(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}
	return result
}

func TestHandleStartIssueWorkPrompt(t *testing.T) {
	s := setupTestServer(t)

	tests := []struct {
		name        string
		args        map[string]interface{}
		wantErr     bool
		checkResult func(t *testing.T, result *mcp.GetPromptResult)
	}{
		{
			name: "minimal args",
			args: map[string]interface{}{
				"issue_number": "123",
			},
			checkResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assert.Contains(t, result.Description, "issue #123")
				assert.Len(t, result.Messages, 2)

				// Check user message
				assert.Equal(t, mcp.RoleUser, result.Messages[0].Role)
				userContent := result.Messages[0].Content.(mcp.TextContent)
				assert.Contains(t, userContent.Text, "issue #123")

				// Check assistant message
				assert.Equal(t, mcp.RoleAssistant, result.Messages[1].Role)
				assistantContent := result.Messages[1].Content.(mcp.TextContent)
				assert.Contains(t, assistantContent.Text, "Requirements Clarification")
				assert.Contains(t, assistantContent.Text, "issue-123")
			},
		},
		{
			name: "with title and URL",
			args: map[string]interface{}{
				"issue_number": "456",
				"issue_title":  "Add new feature",
				"issue_url":    "https://github.com/user/repo/issues/456",
			},
			checkResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assistantContent := result.Messages[1].Content.(mcp.TextContent)
				assert.Contains(t, assistantContent.Text, "Add new feature")
				assert.Contains(t, assistantContent.Text, "https://github.com/user/repo/issues/456")
			},
		},
		{
			name:    "missing issue number",
			args:    map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			request := mcp.GetPromptRequest{
				Params: mcp.GetPromptParams{
					Name:      "start-issue-work",
					Arguments: convertToStringMap(tt.args),
				},
			}

			result, err := s.handleStartIssueWorkPrompt(ctx, request)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestHandlePreparePRPrompt(t *testing.T) {
	s := setupTestServer(t)

	// Create a test workspace
	ws, err := s.workspaceManager.Create(context.Background(), workspace.CreateOptions{
		Name:        "test-pr-workspace",
		Description: "Test workspace for PR prompt",
		BaseBranch:  "main",
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		args        map[string]interface{}
		wantErr     bool
		checkResult func(t *testing.T, result *mcp.GetPromptResult)
	}{
		{
			name: "basic PR preparation",
			args: map[string]interface{}{
				"workspace_id": ws.ID,
			},
			checkResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assert.Contains(t, result.Description, "test-pr-workspace")
				assert.Len(t, result.Messages, 2)

				assistantContent := result.Messages[1].Content.(mcp.TextContent)
				assert.Contains(t, assistantContent.Text, "Pre-PR Checklist")
				assert.Contains(t, assistantContent.Text, "git status")
				assert.Contains(t, assistantContent.Text, "go test")
				assert.Contains(t, assistantContent.Text, ws.Branch)
			},
		},
		{
			name: "with PR details",
			args: map[string]interface{}{
				"workspace_id":   ws.Name, // Test name resolution
				"pr_title":       "feat: add awesome feature",
				"pr_description": "This PR adds an awesome feature",
			},
			checkResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assistantContent := result.Messages[1].Content.(mcp.TextContent)
				assert.Contains(t, assistantContent.Text, "feat: add awesome feature")
			},
		},
		{
			name:    "missing workspace ID",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "invalid workspace ID",
			args: map[string]interface{}{
				"workspace_id": "nonexistent",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			request := mcp.GetPromptRequest{
				Params: mcp.GetPromptParams{
					Name:      "prepare-pr",
					Arguments: convertToStringMap(tt.args),
				},
			}

			result, err := s.handlePreparePRPrompt(ctx, request)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestHandleReviewWorkspacePrompt(t *testing.T) {
	s := setupTestServer(t)

	// Create test workspaces with different ages
	newWs, err := s.workspaceManager.Create(context.Background(), workspace.CreateOptions{
		Name:        "new-workspace",
		Description: "Recently created workspace",
	})
	require.NoError(t, err)

	// Create an "old" workspace (we'll just use the same for testing)
	oldWs, err := s.workspaceManager.Create(context.Background(), workspace.CreateOptions{
		Name:        "old-workspace",
		Description: "Old workspace for testing",
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		args        map[string]interface{}
		wantErr     bool
		checkResult func(t *testing.T, result *mcp.GetPromptResult)
	}{
		{
			name: "review new workspace",
			args: map[string]interface{}{
				"workspace_id": newWs.ID,
			},
			checkResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assert.Contains(t, result.Description, "new-workspace")
				assert.Len(t, result.Messages, 2)

				assistantContent := result.Messages[1].Content.(mcp.TextContent)
				assert.Contains(t, assistantContent.Text, "Workspace Review")
				assert.Contains(t, assistantContent.Text, newWs.ID)
				assert.Contains(t, assistantContent.Text, newWs.Branch)
				assert.Contains(t, assistantContent.Text, "Recently created workspace")
				assert.Contains(t, assistantContent.Text, "git status")
				assert.Contains(t, assistantContent.Text, "Possible Actions")
			},
		},
		{
			name: "review by name",
			args: map[string]interface{}{
				"workspace_id": oldWs.Name,
			},
			checkResult: func(t *testing.T, result *mcp.GetPromptResult) {
				assistantContent := result.Messages[1].Content.(mcp.TextContent)
				assert.Contains(t, assistantContent.Text, "old-workspace")
			},
		},
		{
			name:    "missing workspace ID",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "invalid workspace ID",
			args: map[string]interface{}{
				"workspace_id": "does-not-exist",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			request := mcp.GetPromptRequest{
				Params: mcp.GetPromptParams{
					Name:      "review-workspace",
					Arguments: convertToStringMap(tt.args),
				},
			}

			result, err := s.handleReviewWorkspacePrompt(ctx, request)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestBuildStartIssueWorkPromptText(t *testing.T) {
	s := setupTestServer(t)

	tests := []struct {
		name        string
		issueNumber string
		issueTitle  string
		issueURL    string
		checks      []string
	}{
		{
			name:        "minimal info",
			issueNumber: "42",
			checks: []string{
				"Requirements Clarification",
				"issue-42",
				"What is the problem?",
				"Recommended Workflow",
			},
		},
		{
			name:        "with title",
			issueNumber: "100",
			issueTitle:  "Fix memory leak",
			checks: []string{
				"issue-100",
				"Fix memory leak",
				"description: \"Fix memory leak\"",
			},
		},
		{
			name:        "with URL",
			issueNumber: "200",
			issueURL:    "https://github.com/user/repo/issues/200",
			checks: []string{
				"https://github.com/user/repo/issues/200",
				"📎 Issue URL:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := s.buildStartIssueWorkPromptText(tt.issueNumber, tt.issueTitle, tt.issueURL)

			for _, check := range tt.checks {
				assert.Contains(t, text, check)
			}

			// Check structure
			assert.True(t, strings.Contains(text, "## 🔍"))
			assert.True(t, strings.Contains(text, "## 📋"))
			assert.True(t, strings.Contains(text, "## 💡"))
		})
	}
}

func TestBuildPreparePRPromptText(t *testing.T) {
	s := setupTestServer(t)

	ws := &workspace.Workspace{
		ID:         "ws-123",
		Name:       "feature-branch",
		Branch:     "feature/awesome",
		BaseBranch: "develop",
	}

	tests := []struct {
		name          string
		prTitle       string
		prDescription string
		checks        []string
	}{
		{
			name: "without PR details",
			checks: []string{
				"feature-branch",
				"Pre-PR Checklist",
				"git status",
				"go test ./...",
				"develop",
				"feature/awesome",
				"Title",
				"Description",
			},
		},
		{
			name:    "with title",
			prTitle: "feat: amazing feature",
			checks: []string{
				"feat: amazing feature",
				"--title \"feat: amazing feature\"",
			},
		},
		{
			name:          "with description",
			prDescription: "This PR does amazing things",
			checks: []string{
				"Pre-PR Checklist",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := s.buildPreparePRPromptText(ws, tt.prTitle, tt.prDescription)

			for _, check := range tt.checks {
				assert.Contains(t, text, check)
			}

			// Check commands are present
			assert.Contains(t, text, "git push -u origin")
			assert.Contains(t, text, "gh pr create --draft")
		})
	}
}

func TestBuildReviewWorkspacePromptText(t *testing.T) {
	s := setupTestServer(t)

	ws := &workspace.Workspace{
		ID:          "ws-review",
		Name:        "review-test",
		Branch:      "feature/review",
		BaseBranch:  "main",
		Description: "Test workspace for review",
		CreatedAt:   time.Now().Add(-2 * time.Hour),
		UpdatedAt:   time.Now(),
	}

	text := s.buildReviewWorkspacePromptText(ws)

	// Check all sections are present
	assert.Contains(t, text, "Workspace Review: review-test")
	assert.Contains(t, text, "📋 Workspace Details")
	assert.Contains(t, text, "🎯 Recommended Next Steps")
	assert.Contains(t, text, "💭 Questions to Consider")
	assert.Contains(t, text, "🔄 Possible Actions")

	// Check workspace details
	assert.Contains(t, text, ws.ID)
	assert.Contains(t, text, ws.Branch)
	assert.Contains(t, text, ws.BaseBranch)
	assert.Contains(t, text, ws.Description)

	// Check age calculation - the Age field is included in the string
	assert.Contains(t, text, "**Age**:")
	assert.Contains(t, text, "hours")

	// Check resource URIs
	assert.Contains(t, text, "amux://workspace/"+ws.ID+"/context")
	assert.Contains(t, text, "amux://workspace/"+ws.ID+"/files")

	// Check commands
	assert.Contains(t, text, "git status")
	assert.Contains(t, text, "git diff main...HEAD")
}

func TestRegisterPrompts(t *testing.T) {
	s := setupTestServer(t)

	// Test that registerPrompts completes without error
	// Since MCPServer doesn't expose ListPrompts method,
	// we'll verify by testing that the handlers work

	ctx := context.Background()

	// Test start-issue-work prompt exists
	startReq := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "start-issue-work",
			Arguments: map[string]string{
				"issue_number": "1",
			},
		},
	}
	result, err := s.handleStartIssueWorkPrompt(ctx, startReq)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Test prepare-pr prompt exists
	ws, err := s.workspaceManager.Create(context.Background(), workspace.CreateOptions{
		Name: "test-prompt-reg",
	})
	require.NoError(t, err)

	prReq := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "prepare-pr",
			Arguments: map[string]string{
				"workspace_id": ws.ID,
			},
		},
	}
	result, err = s.handlePreparePRPrompt(ctx, prReq)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Test review-workspace prompt exists
	reviewReq := mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "review-workspace",
			Arguments: map[string]string{
				"workspace_id": ws.ID,
			},
		},
	}
	result, err = s.handleReviewWorkspacePrompt(ctx, reviewReq)
	require.NoError(t, err)
	assert.NotNil(t, result)
}
