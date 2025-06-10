package session

import "fmt"

// ErrSessionNotFound is returned when a session cannot be found
type ErrSessionNotFound struct {
	ID string
}

func (e ErrSessionNotFound) Error() string {
	return fmt.Sprintf("session not found: %s", e.ID)
}

// ErrSessionAlreadyRunning is returned when trying to start a running session
type ErrSessionAlreadyRunning struct {
	ID string
}

func (e ErrSessionAlreadyRunning) Error() string {
	return fmt.Sprintf("session already running: %s", e.ID)
}

// ErrSessionNotRunning is returned when trying to interact with a non-running session
type ErrSessionNotRunning struct {
	ID string
}

func (e ErrSessionNotRunning) Error() string {
	return fmt.Sprintf("session not running: %s", e.ID)
}

// ErrInvalidWorkspace is returned when the workspace doesn't exist
type ErrInvalidWorkspace struct {
	WorkspaceID string
}

func (e ErrInvalidWorkspace) Error() string {
	return fmt.Sprintf("invalid workspace: %s", e.WorkspaceID)
}

// ErrInvalidAgent is returned when the agent doesn't exist
type ErrInvalidAgent struct {
	AgentID string
}

func (e ErrInvalidAgent) Error() string {
	return fmt.Sprintf("invalid agent: %s", e.AgentID)
}

// ErrTmuxNotAvailable is returned when tmux is not installed or accessible
type ErrTmuxNotAvailable struct{}

func (e ErrTmuxNotAvailable) Error() string {
	return "tmux is not available on this system"

}
