package state

// Status represents the lifecycle state of a session
type Status string

// Status constants define all possible session states
const (
	// StatusCreated indicates the session has been created but not started
	StatusCreated Status = "created"

	// StatusStarting indicates the session is in the process of starting
	StatusStarting Status = "starting"

	// StatusRunning indicates the session is actively running
	StatusRunning Status = "running"

	// StatusStopping indicates the session is in the process of stopping
	StatusStopping Status = "stopping"

	// StatusCompleted indicates the session completed successfully
	StatusCompleted Status = "completed"

	// StatusStopped indicates the session was stopped by user request
	StatusStopped Status = "stopped"

	// StatusFailed indicates the session failed due to an error
	StatusFailed Status = "failed"

	// StatusOrphaned indicates the session's workspace no longer exists
	StatusOrphaned Status = "orphaned"
)

// String returns the string representation of the status
func (s Status) String() string {
	return string(s)
}

// IsTerminal returns true if the status represents a terminal state
func (s Status) IsTerminal() bool {
	switch s {
	case StatusCompleted, StatusStopped, StatusFailed, StatusOrphaned:
		return true
	case StatusCreated, StatusStarting, StatusRunning, StatusStopping:
		return false
	}
	// This should never be reached if all Status values are handled
	return false
}

// IsRunning returns true if the status represents an active state
func (s Status) IsRunning() bool {
	switch s {
	case StatusStarting, StatusRunning, StatusStopping:
		return true
	case StatusCreated, StatusCompleted, StatusStopped, StatusFailed, StatusOrphaned:
		return false
	}
	// This should never be reached if all Status values are handled
	return false
}

// allowedTransitions defines the valid state transitions
var allowedTransitions = map[Status][]Status{
	StatusCreated:  {StatusStarting, StatusFailed, StatusOrphaned},
	StatusStarting: {StatusRunning, StatusFailed, StatusOrphaned},
	StatusRunning:  {StatusStopping, StatusCompleted, StatusFailed, StatusOrphaned},
	StatusStopping: {StatusStopped, StatusFailed, StatusOrphaned},
	// Terminal states have no valid transitions
	StatusCompleted: {},
	StatusStopped:   {},
	StatusFailed:    {},
	StatusOrphaned:  {},
}

// CanTransitionTo checks if a transition to the target status is allowed
func (s Status) CanTransitionTo(target Status) bool {
	allowed, exists := allowedTransitions[s]
	if !exists {
		return false
	}

	for _, validTarget := range allowed {
		if validTarget == target {
			return true
		}
	}
	return false
}

// ValidateTransition returns an error if the transition is not allowed
func ValidateTransition(from, to Status) error {
	if !from.CanTransitionTo(to) {
		return &ErrInvalidTransition{
			From: from,
			To:   to,
		}
	}
	return nil
}
