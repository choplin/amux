package state

import (
	"testing"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusCreated, "created"},
		{StatusStarting, "starting"},
		{StatusRunning, "running"},
		{StatusStopping, "stopping"},
		{StatusCompleted, "completed"},
		{StatusStopped, "stopped"},
		{StatusFailed, "failed"},
		{StatusOrphaned, "orphaned"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("Status.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   Status
		terminal bool
	}{
		{StatusCreated, false},
		{StatusStarting, false},
		{StatusRunning, false},
		{StatusStopping, false},
		{StatusCompleted, true},
		{StatusStopped, true},
		{StatusFailed, true},
		{StatusOrphaned, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.terminal {
				t.Errorf("Status.IsTerminal() = %v, want %v", got, tt.terminal)
			}
		})
	}
}

func TestStatus_IsRunning(t *testing.T) {
	tests := []struct {
		status  Status
		running bool
	}{
		{StatusCreated, false},
		{StatusStarting, true},
		{StatusRunning, true},
		{StatusStopping, true},
		{StatusCompleted, false},
		{StatusStopped, false},
		{StatusFailed, false},
		{StatusOrphaned, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsRunning(); got != tt.running {
				t.Errorf("Status.IsRunning() = %v, want %v", got, tt.running)
			}
		})
	}
}

func TestStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name     string
		from     Status
		to       Status
		expected bool
	}{
		// Created transitions
		{"created to starting", StatusCreated, StatusStarting, true},
		{"created to failed", StatusCreated, StatusFailed, true},
		{"created to orphaned", StatusCreated, StatusOrphaned, true},
		{"created to running", StatusCreated, StatusRunning, false},
		{"created to completed", StatusCreated, StatusCompleted, false},

		// Starting transitions
		{"starting to running", StatusStarting, StatusRunning, true},
		{"starting to failed", StatusStarting, StatusFailed, true},
		{"starting to orphaned", StatusStarting, StatusOrphaned, true},
		{"starting to stopped", StatusStarting, StatusStopped, false},

		// Running transitions
		{"running to stopping", StatusRunning, StatusStopping, true},
		{"running to completed", StatusRunning, StatusCompleted, true},
		{"running to failed", StatusRunning, StatusFailed, true},
		{"running to orphaned", StatusRunning, StatusOrphaned, true},
		{"running to starting", StatusRunning, StatusStarting, false},

		// Stopping transitions
		{"stopping to stopped", StatusStopping, StatusStopped, true},
		{"stopping to failed", StatusStopping, StatusFailed, true},
		{"stopping to orphaned", StatusStopping, StatusOrphaned, true},
		{"stopping to running", StatusStopping, StatusRunning, false},

		// Terminal states cannot transition
		{"completed to any", StatusCompleted, StatusRunning, false},
		{"stopped to any", StatusStopped, StatusRunning, false},
		{"failed to any", StatusFailed, StatusRunning, false},
		{"orphaned to any", StatusOrphaned, StatusRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.expected {
				t.Errorf("CanTransitionTo(%v, %v) = %v, want %v", tt.from, tt.to, got, tt.expected)
			}
		})
	}
}

func TestValidateTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    Status
		to      Status
		wantErr bool
	}{
		{"valid transition", StatusCreated, StatusStarting, false},
		{"invalid transition", StatusCreated, StatusStopped, true},
		{"terminal state", StatusCompleted, StatusRunning, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransition(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTransition() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && !tt.wantErr {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
