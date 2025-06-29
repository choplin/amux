// Package test provides integration tests for amux functionality.
package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
	runtimeinit "github.com/aki/amux/internal/runtime/init"
	"github.com/aki/amux/internal/tests/helpers"
)

func TestMain(m *testing.M) {
	InitTestLogger()
	os.Exit(m.Run())
}

func TestIntegration_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Initialize runtime registry for tests
	if err := runtimeinit.RegisterDefaults(); err != nil {
		t.Fatalf("Failed to initialize runtimes: %v", err)
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
	cfg.Agents["test-agent"] = config.Agent{
		Name:    "Test Agent",
		Runtime: "tmux",
		Environment: map[string]string{
			"TEST_ENV": "integration",
		},
		Command: []string{"echo", "Test agent running"},
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
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name:        "integration-test",
		BaseBranch:  "main",
		Description: "Integration test workspace",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Verify workspace was created
	if _, err := os.Stat(ws.Path); err != nil {
		t.Errorf("Workspace path does not exist: %v", err)
	}

	// Context functionality has been deprecated in favor of storage directories

	// Create session manager with mock adapter
	sessionManager, err := session.NewManager(configManager.GetAmuxDir(), wsManager, configManager, nil)
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	// Use mock adapter for predictable testing
	mockAdapter := tmux.NewMockAdapter()
	sessionManager.SetTmuxAdapter(mockAdapter)

	// Get agent configuration (for debugging if needed)
	_, _ = configManager.GetAgent("test-agent")

	// Get command and environment from agent config
	command := "test-command"
	env := make(map[string]string)

	// Create a session with agent configuration
	opts := session.Options{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Command:     command,
		Environment: env,
	}

	sess, err := sessionManager.CreateSession(context.Background(), opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Start the session
	ctx := context.Background()
	if err := sess.Start(ctx); err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Verify session is running (either working or idle)
	if !sess.Status().IsRunning() {
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
		}
	}

	// Type assert to TerminalSession for terminal operations
	terminalSess, ok := sess.(session.TerminalSession)
	if !ok {
		t.Fatal("Session does not support terminal operations")
	}

	// Send some input
	if err := terminalSess.SendInput(context.Background(), "echo 'Integration test complete'"); err != nil {
		t.Errorf("Failed to send input: %v", err)
	}

	// Get output
	output, err := terminalSess.GetOutput(0)
	if err != nil {
		t.Errorf("Failed to get output: %v", err)
	}

	if len(output) == 0 {
		t.Error("Expected some output from session")
	}

	// Stop the session
	if err := sess.Stop(context.Background()); err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}

	// List all sessions
	sessions, err := sessionManager.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	// Clean up workspace
	if err := wsManager.Remove(context.Background(), workspace.Identifier(ws.ID), workspace.RemoveOptions{}); err != nil {
		t.Errorf("Failed to remove workspace: %v", err)
	}
}

func TestIntegration_MultipleAgents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Initialize runtime registry for tests
	if err := runtimeinit.RegisterDefaults(); err != nil {
		t.Fatalf("Failed to initialize runtimes: %v", err)
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
		Runtime: "tmux",
		Command: []string{"echo", "Agent 1"},
	}
	cfg.Agents["agent2"] = config.Agent{
		Name:    "Agent 2",
		Runtime: "tmux",
		Command: []string{"echo", "Agent 2"},
	}
	cfg.Agents["agent3"] = config.Agent{
		Name:    "Agent 3",
		Runtime: "tmux",
		Command: []string{"echo", "Agent 3"},
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
	sessionManager, err := session.NewManager(configManager.GetAmuxDir(), wsManager, configManager, nil)
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}
	mockAdapter := tmux.NewMockAdapter()
	sessionManager.SetTmuxAdapter(mockAdapter)

	// Create workspaces and sessions for each agent
	var sessions []session.Session
	agents := []string{"agent1", "agent2", "agent3"}

	for _, agentID := range agents {
		// Create workspace
		ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
			Name: "workspace-" + agentID,
		})
		if err != nil {
			t.Fatalf("Failed to create workspace for %s: %v", agentID, err)
		}

		// Create session
		sess, err := sessionManager.CreateSession(context.Background(), session.Options{
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
	allSessions, err := sessionManager.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(allSessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(allSessions))
	}

	// Send different input to each session
	for i, sess := range sessions {
		// Type assert to TerminalSession
		terminalSess, ok := sess.(session.TerminalSession)
		if !ok {
			t.Errorf("Session %d does not support terminal operations", i)
			continue
		}
		input := "echo 'Output from session " + sess.ID() + "'"
		if err := terminalSess.SendInput(context.Background(), input); err != nil {
			t.Errorf("Failed to send input to session %d: %v", i, err)
		}
	}

	// Stop all sessions
	for _, sess := range sessions {
		if err := sess.Stop(context.Background()); err != nil {
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
