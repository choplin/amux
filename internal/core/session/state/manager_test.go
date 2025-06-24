package state

import (
	"context"
	"testing"
	"time"
)

// mockLogger implements the Logger interface for testing
type mockLogger struct {
	messages []string
}

func (m *mockLogger) Debug(msg string, args ...interface{}) {
	m.messages = append(m.messages, "DEBUG: "+msg)
}

func (m *mockLogger) Info(msg string, args ...interface{}) {
	m.messages = append(m.messages, "INFO: "+msg)
}

func (m *mockLogger) Warn(msg string, args ...interface{}) {
	m.messages = append(m.messages, "WARN: "+msg)
}

func (m *mockLogger) Error(msg string, args ...interface{}) {
	m.messages = append(m.messages, "ERROR: "+msg)
}

func TestManager_CurrentState(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &mockLogger{}
	manager := NewManager("session-123", "workspace-456", tmpDir, logger)

	// Test default state (no file exists)
	state, err := manager.CurrentState()
	if err != nil {
		t.Fatalf("CurrentState() error = %v", err)
	}
	if state != StatusCreated {
		t.Errorf("CurrentState() = %v, want %v", state, StatusCreated)
	}

	// Create a state file
	stateData := &StateData{
		Status:           StatusRunning,
		StatusChangedAt:  time.Now(),
		LastActivityTime: time.Now(),
	}
	if err := manager.saveState(stateData); err != nil {
		t.Fatalf("saveState() error = %v", err)
	}

	// Test reading existing state
	state, err = manager.CurrentState()
	if err != nil {
		t.Fatalf("CurrentState() error = %v", err)
	}
	if state != StatusRunning {
		t.Errorf("CurrentState() = %v, want %v", state, StatusRunning)
	}
}

func TestManager_TransitionTo(t *testing.T) {
	tests := []struct {
		name         string
		initialState Status
		targetState  Status
		wantErr      bool
		errorType    interface{}
	}{
		{
			name:         "valid transition created to starting",
			initialState: StatusCreated,
			targetState:  StatusStarting,
			wantErr:      false,
		},
		{
			name:         "valid transition starting to running",
			initialState: StatusStarting,
			targetState:  StatusRunning,
			wantErr:      false,
		},
		{
			name:         "invalid transition created to stopped",
			initialState: StatusCreated,
			targetState:  StatusStopped,
			wantErr:      true,
			errorType:    &ErrInvalidTransition{},
		},
		{
			name:         "already in state",
			initialState: StatusRunning,
			targetState:  StatusRunning,
			wantErr:      true,
			errorType:    &ErrAlreadyInState{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logger := &mockLogger{}
			manager := NewManager("session-123", "workspace-456", tmpDir, logger)

			// Set initial state if not created
			if tt.initialState != StatusCreated {
				// Set up the state through valid transitions
				switch tt.initialState {
				case StatusStarting:
					_ = manager.TransitionTo(context.Background(), StatusStarting)
				case StatusRunning:
					_ = manager.TransitionTo(context.Background(), StatusStarting)
					_ = manager.TransitionTo(context.Background(), StatusRunning)
				case StatusStopping:
					_ = manager.TransitionTo(context.Background(), StatusStarting)
					_ = manager.TransitionTo(context.Background(), StatusRunning)
					_ = manager.TransitionTo(context.Background(), StatusStopping)
				}
			}

			// Attempt the transition
			err := manager.TransitionTo(context.Background(), tt.targetState)

			if (err != nil) != tt.wantErr {
				t.Errorf("TransitionTo() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errorType != nil {
				switch tt.errorType.(type) {
				case *ErrInvalidTransition:
					if _, ok := err.(*ErrInvalidTransition); !ok {
						t.Errorf("Expected ErrInvalidTransition, got %T", err)
					}
				case *ErrAlreadyInState:
					if _, ok := err.(*ErrAlreadyInState); !ok {
						t.Errorf("Expected ErrAlreadyInState, got %T", err)
					}
				}
			}

			// Verify state if transition was successful
			if !tt.wantErr {
				currentState, _ := manager.CurrentState()
				if currentState != tt.targetState {
					t.Errorf("After transition, state = %v, want %v", currentState, tt.targetState)
				}
			}
		})
	}
}

func TestManager_StateChangeHandlers(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &mockLogger{}
	manager := NewManager("session-123", "workspace-456", tmpDir, logger)

	// Track handler calls
	handlerCalls := make([]string, 0)

	// Add handlers
	handler1 := func(ctx context.Context, from, to Status, sessionID, workspaceID string) error {
		handlerCalls = append(handlerCalls, "handler1")
		return nil
	}

	handler2 := func(ctx context.Context, from, to Status, sessionID, workspaceID string) error {
		handlerCalls = append(handlerCalls, "handler2")
		return nil
	}

	manager.AddStateChangeHandler(handler1)
	manager.AddStateChangeHandler(handler2)

	// Perform transition
	err := manager.TransitionTo(context.Background(), StatusStarting)
	if err != nil {
		t.Fatalf("TransitionTo() error = %v", err)
	}

	// Verify handlers were called
	if len(handlerCalls) != 2 {
		t.Errorf("Expected 2 handler calls, got %d", len(handlerCalls))
	}

	if len(handlerCalls) >= 2 {
		if handlerCalls[0] != "handler1" || handlerCalls[1] != "handler2" {
			t.Errorf("Handlers called in wrong order: %v", handlerCalls)
		}
	}
}

func TestManager_UpdateActivity(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &mockLogger{}
	manager := NewManager("session-123", "workspace-456", tmpDir, logger)

	// Transition to running state
	_ = manager.TransitionTo(context.Background(), StatusStarting)
	_ = manager.TransitionTo(context.Background(), StatusRunning)

	// Get initial state data
	data1, err := manager.StateData()
	if err != nil {
		t.Fatalf("StateData() error = %v", err)
	}

	// Sleep briefly to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update activity
	err = manager.UpdateActivity()
	if err != nil {
		t.Fatalf("UpdateActivity() error = %v", err)
	}

	// Get updated state data
	data2, err := manager.StateData()
	if err != nil {
		t.Fatalf("StateData() error = %v", err)
	}

	// Verify activity time was updated but status wasn't changed
	if data2.Status != data1.Status {
		t.Errorf("Status changed unexpectedly: %v -> %v", data1.Status, data2.Status)
	}

	if !data2.LastActivityTime.After(data1.LastActivityTime) {
		t.Errorf("LastActivityTime not updated: %v -> %v", data1.LastActivityTime, data2.LastActivityTime)
	}

	if data2.StatusChangedAt != data1.StatusChangedAt {
		t.Errorf("StatusChangedAt changed unexpectedly: %v -> %v", data1.StatusChangedAt, data2.StatusChangedAt)
	}
}

func TestManager_StatePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &mockLogger{}

	// Create first manager and set state
	manager1 := NewManager("session-123", "workspace-456", tmpDir, logger)
	_ = manager1.TransitionTo(context.Background(), StatusStarting)
	_ = manager1.TransitionTo(context.Background(), StatusRunning)

	// Create second manager with same paths
	manager2 := NewManager("session-123", "workspace-456", tmpDir, logger)

	// Verify state is persisted
	state, err := manager2.CurrentState()
	if err != nil {
		t.Fatalf("CurrentState() error = %v", err)
	}

	if state != StatusRunning {
		t.Errorf("Persisted state = %v, want %v", state, StatusRunning)
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	logger := &mockLogger{}
	manager := NewManager("session-123", "workspace-456", tmpDir, logger)

	// Run concurrent transitions
	done := make(chan bool, 3)

	// Goroutine 1: Transition to starting
	go func() {
		_ = manager.TransitionTo(context.Background(), StatusStarting)
		done <- true
	}()

	// Goroutine 2: Read current state
	go func() {
		_, _ = manager.CurrentState()
		done <- true
	}()

	// Goroutine 3: Update activity
	go func() {
		_ = manager.UpdateActivity()
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify file is not corrupted
	state, err := manager.CurrentState()
	if err != nil {
		t.Fatalf("CurrentState() after concurrent access error = %v", err)
	}

	// State should be either created or starting
	if state != StatusCreated && state != StatusStarting {
		t.Errorf("Unexpected state after concurrent access: %v", state)
	}
}
