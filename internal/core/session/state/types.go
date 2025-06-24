package state

import (
	"encoding/json"
	"fmt"
	"time"
)

// Status represents the lifecycle state of a session
type Status string

const (
	// StatusCreated indicates the session has been created but not started
	StatusCreated Status = "created"
	// StatusStarting indicates the session is in the process of starting
	StatusStarting Status = "starting"
	// StatusRunning indicates the session is running (generic running state)
	StatusRunning Status = "running"
	// StatusWorking indicates the session is actively processing (agent is working)
	StatusWorking Status = "working"
	// StatusIdle indicates the session is running but inactive (agent is idle)
	StatusIdle Status = "idle"
	// StatusStopping indicates the session is in the process of stopping
	StatusStopping Status = "stopping"
	// StatusStopped indicates the session has been stopped by user
	StatusStopped Status = "stopped"
	// StatusFailed indicates the session failed to start or crashed
	StatusFailed Status = "failed"
	// StatusCompleted indicates the session completed successfully
	StatusCompleted Status = "completed"
	// StatusOrphaned indicates the session's workspace was deleted
	StatusOrphaned Status = "orphaned"
)

// IsRunning returns true if the status indicates the session is running
func (s Status) IsRunning() bool {
	return s == StatusRunning || s == StatusWorking || s == StatusIdle
}

// IsTerminal returns true if the status is a terminal state
func (s Status) IsTerminal() bool {
	return s == StatusStopped || s == StatusFailed || s == StatusCompleted || s == StatusOrphaned
}

// Data represents the persistent state of a session
type Data struct {
	State       Status    `json:"state"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   int       `json:"updated_by"`
	SessionID   string    `json:"session_id"`
	WorkspaceID string    `json:"workspace_id"`

	// Activity tracking
	LastOutputHash  uint32    `json:"last_output_hash,omitempty"`
	LastOutputTime  time.Time `json:"last_output_time,omitempty"`
	LastStatusCheck time.Time `json:"last_status_check,omitempty"`
}

// ErrSessionLocked is returned when a session is locked by another process
type ErrSessionLocked struct {
	SessionID string
	LockedBy  *LockInfo
}

func (e *ErrSessionLocked) Error() string {
	if e.LockedBy != nil {
		return fmt.Sprintf("session %s is locked by process %d (%s) since %v",
			e.SessionID, e.LockedBy.PID, e.LockedBy.Operation, e.LockedBy.AcquiredAt)
	}
	return fmt.Sprintf("session %s is locked", e.SessionID)
}

// LockInfo contains information about who holds a lock
type LockInfo struct {
	PID        int       `json:"pid"`
	Operation  string    `json:"operation"`
	AcquiredAt time.Time `json:"acquired_at"`
}

// LockType represents the type of lock (internal type)
type LockType string

const (
	// ReadLock is a shared lock for read operations
	ReadLock LockType = "read"
	// WriteLock is an exclusive lock for write operations
	WriteLock LockType = "write"
)

// Lock represents a lock that can be released (internal type)
type Lock interface {
	Release() error
}

// MarshalJSON implements json.Marshaler for Status
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

// UnmarshalJSON implements json.Unmarshaler for Status
func (s *Status) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*s = Status(str)
	return nil
}
