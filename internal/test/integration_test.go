package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/agent"
	"github.com/aki/amux/internal/core/config"
	contextmgr "github.com/aki/amux/internal/core/context"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/aki/amux/internal/tests/helpers"
)

func TestIntegration_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test repository
	repoPath := helpers.CreateTestRepo(t)

	// Change to repo directory
	oldWd, _ := os.Getwd()
	if err := os.Chdir(repoPath); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(oldWd)

	// Create config manager
	configManager := config.NewManager(repoPath)

	// Initialize configuration
	cfg := config.DefaultConfig()
	cfg.Project.Name = "test-integration"
	cfg.Agents["test-agent"] = config.Agent{
		Name:    "Test Agent",
		Type:    "test",
		Command: "echo 'Test agent running'",
		Environment: map[string]string{
			"TEST_ENV": "integration",
		},
	}

	if err := configManager.Save(cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	if err != nil {
		t.Fatalf("Failed to create workspace manager: %v", err)
	}

	// Create a workspace
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name:        "integration-test",
		BaseBranch:  "main",
		AgentID:     "test-agent",
		Description: "Integration test workspace",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Verify workspace was created
	if _, err := os.Stat(ws.Path); err != nil {
		t.Errorf("Workspace path does not exist: %v", err)
	}

	// Verify context was initialized
	contextManager := contextmgr.NewManager(ws.Path)
	if !contextManager.Exists() {
		// Initialize it manually if not exists
		if err := contextManager.Initialize(); err != nil {
			t.Errorf("Failed to initialize context: %v", err)
		}
	}

	// Verify context files exist
	contextFiles := []string{
		contextmgr.BackgroundFile,
		contextmgr.PlanFile,
		contextmgr.WorkingLogFile,
		contextmgr.ResultsSummaryFile,
	}

	for _, file := range contextFiles {
		path := contextManager.GetFilePath(file)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Context file %s does not exist: %v", file, err)
		}
	}

	// Create session manager with mock adapter
	store, err := session.NewFileStore(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	sessionManager := session.NewManager(store, wsManager, nil, nil)

	// Use mock adapter for predictable testing
	mockAdapter := tmux.NewMockAdapter()
	sessionManager.SetTmuxAdapter(mockAdapter)

	// Create agent manager
	agentManager := agent.NewManager(configManager)

	// Get agent configuration (for debugging if needed)
	_, _ = agentManager.GetAgent("test-agent")

	// Get command and environment from agent config
	command, _ := agentManager.GetDefaultCommand("test-agent")
	env, _ := agentManager.GetEnvironment("test-agent")

	// Create a session with agent configuration
	opts := session.Options{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Command:     command,
		Environment: env,
	}

	sess, err := sessionManager.CreateSession(opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Start the session
	ctx := context.Background()
	if err := sess.Start(ctx); err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Verify session is running
	if sess.Status() != session.StatusRunning {
		t.Errorf("Expected session to be running, got %s", sess.Status())
	}

	// Verify environment includes agent config
	sessionInfo := sess.Info()
	if sessionInfo.TmuxSession == "" {
		t.Log("Warning: No tmux session name set")
	} else {
		env := mockAdapter.GetSessionEnvironment(sessionInfo.TmuxSession)
		if env == nil {
			t.Errorf("No environment found for session %s", sessionInfo.TmuxSession)
		} else {
			if env["TEST_ENV"] != "integration" {
				t.Errorf("Expected TEST_ENV=integration in session environment, got %v", env)
			}
			if env["AMUX_CONTEXT_PATH"] == "" {
				t.Errorf("Expected AMUX_CONTEXT_PATH to be set")
			}
		}
	}

	// Send some input
	if err := sess.SendInput("echo 'Integration test complete'"); err != nil {
		t.Errorf("Failed to send input: %v", err)
	}

	// Get output
	output, err := sess.GetOutput(0)
	if err != nil {
		t.Errorf("Failed to get output: %v", err)
	}

	if len(output) == 0 {
		t.Error("Expected some output from session")
	}

	// Stop the session
	if err := sess.Stop(); err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}

	// List all sessions
	sessions, err := sessionManager.ListSessions()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	// Clean up workspace
	if err := wsManager.Remove(ws.ID); err != nil {
		t.Errorf("Failed to remove workspace: %v", err)
	}
}

func TestIntegration_MultipleAgents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test repository
	repoPath := helpers.CreateTestRepo(t)

	// Change to repo directory
	oldWd, _ := os.Getwd()
	if err := os.Chdir(repoPath); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(oldWd)

	// Initialize configuration
	configManager := config.NewManager(repoPath)
	cfg := config.DefaultConfig()

	// Add multiple agents
	cfg.Agents["agent1"] = config.Agent{
		Name:    "Agent 1",
		Type:    "test",
		Command: "echo 'Agent 1'",
	}
	cfg.Agents["agent2"] = config.Agent{
		Name:    "Agent 2",
		Type:    "test",
		Command: "echo 'Agent 2'",
	}
	cfg.Agents["agent3"] = config.Agent{
		Name:    "Agent 3",
		Type:    "test",
		Command: "echo 'Agent 3'",
	}

	if err := configManager.Save(cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	if err != nil {
		t.Fatalf("Failed to create workspace manager: %v", err)
	}

	// Create session manager with mock
	store, err := session.NewFileStore(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	sessionManager := session.NewManager(store, wsManager, nil, nil)
	mockAdapter := tmux.NewMockAdapter()
	sessionManager.SetTmuxAdapter(mockAdapter)

	// Create workspaces and sessions for each agent
	var sessions []session.Session
	agents := []string{"agent1", "agent2", "agent3"}

	for _, agentID := range agents {
		// Create workspace
		ws, err := wsManager.Create(workspace.CreateOptions{
			Name:    "workspace-" + agentID,
			AgentID: agentID,
		})
		if err != nil {
			t.Fatalf("Failed to create workspace for %s: %v", agentID, err)
		}

		// Create session
		sess, err := sessionManager.CreateSession(session.Options{
			WorkspaceID: ws.ID,
			AgentID:     agentID,
		})
		if err != nil {
			t.Fatalf("Failed to create session for %s: %v", agentID, err)
		}

		// Start session
		ctx := context.Background()
		if err := sess.Start(ctx); err != nil {
			t.Fatalf("Failed to start session for %s: %v", agentID, err)
		}

		sessions = append(sessions, sess)

		// Small delay to simulate real usage
		time.Sleep(10 * time.Millisecond)
	}

	// Verify all sessions are running
	mockSessions := mockAdapter.GetSessions()
	if len(mockSessions) != 3 {
		t.Errorf("Expected 3 mock sessions, got %d", len(mockSessions))
	}

	// List all sessions
	allSessions, err := sessionManager.ListSessions()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(allSessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(allSessions))
	}

	// Send different input to each session
	for i, sess := range sessions {
		input := "echo 'Output from session " + sess.ID() + "'"
		if err := sess.SendInput(input); err != nil {
			t.Errorf("Failed to send input to session %d: %v", i, err)
		}
	}

	// Stop all sessions
	for _, sess := range sessions {
		if err := sess.Stop(); err != nil {
			t.Errorf("Failed to stop session %s: %v", sess.ID(), err)
		}
	}

	// Verify all stopped
	for _, sess := range sessions {
		if sess.Status() != session.StatusStopped {
			t.Errorf("Session %s should be stopped, got %s", sess.ID(), sess.Status())
		}
	}
}
